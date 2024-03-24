package data

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Store interface {
	// Cleanup cleans the underlying connection to the data store.
	Cleanup(ctx context.Context) error

	// IsRateLimited returns true if the given IP is being rate limited.
	IsRateLimited(identifier string, resource string) (bool, error)

	// RateLimit marks IP & Resource as rate limited for n ms.
	RateLimit(identifier string, resource string, ms int) error

	// WriteCategory adds a new category to the database.
	WriteCategory(ctx context.Context, categoryTag string, categoryName string) error

	/*
		RemoveCategory removes all posts under category categoryTag and removes the category.
		Returns affected rows.
	*/
	RemoveCategory(ctx context.Context, categoryTag string) (int64, error)

	// GetThreadCount returns the number of threads in a category.
	GetThreadCount(ctx context.Context, categoryTag string) (int, error)

	// GetCategories returns all categories.
	GetCategories(ctx context.Context) ([]*Category, error)

	/*
		GetPostByNumber returns a post in a category by its number.
		Should return ErrNotFound if no such post.
	*/
	GetPostByNumber(ctx context.Context, categoryTag string, num int) (*Post, error)

	/*
		GetThreadView returns all the posts in a thread, and the category they're on.
		Should return ErrNotFound if the requested thread is not an OP thread, or the category
		is invalid
	*/
	GetThreadView(ctx context.Context, categoryTag string, threadNum int) (*ThreadView, error)

	/*
		GetCategory returns a single category. May return ErrNotFound if the given category
		name is invalid.
	*/
	GetCategory(ctx context.Context, categoryTag string) (*Category, error)

	/*
		GetCategoryView returns information about a category, and all the threads on it.
		May return an ErrNotFound if the given category name is invalid.
	*/
	GetCategoryView(ctx context.Context, categoryTag string) (*CatView, error)

	/*
		Creates a post.
		Optional parent thread can be provided if it's a reply.
		Should return ErrNotFound if invalid post or category.
	*/
	WritePost(ctx context.Context, categoryTag string, parentThreadNumber int, subject string, content string, username string, email string, ip string) error

	/*
		Removes a post at the given category & number.
		Returns number of rows affected.
	*/
	RemovePost(ctx context.Context, categoryTag string, number int) (int, error)
}

var ErrNotFound = errors.New("not found")

// Returns a string identifying a resource and a rate limit identifier (IP addr usually)
func getRateLimitResourceID(identifier string, resource string) string {
	return fmt.Sprintf("%s-%s", identifier, resource)
}

// Category contains JSON information describing a Category for posts.
type Category struct {
	Tag         string `json:"tag"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PostCount   int    `json:"postCount"`
}

// Post contains JSON information describing a thread, or reply to a thread.
type Post struct {
	Num       int       `json:"num"`
	Cat       string    `json:"cat"`
	Parent    int       `json:"-"`
	Subject   string    `json:"subject"`
	Content   string    `json:"content"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"createdAt"`
}

// UserPost contains JSON information describing an incoming post for writing.
type UserPost struct {
	Content string `json:"content"`
	Subject string `json:"subject"`
}

// IsReply returns true if this post has a parent.
func (post Post) IsReply() bool {
	return post.Parent != 0
}

// CatView contains JSON information about a category, and all the threads on it.
type CatView struct {
	Category *Category `json:"category"`
	Threads  []*Post   `json:"threads"`
}

/*
ThreadView contains JSON information about all
the posts in a thread, and the category its on.
*/
type ThreadView struct {
	Category *Category `json:"category"`
	Posts    []*Post   `json:"posts"`
}

// NewDatastore creates a new data store, creating a connection.
func NewDatastore(ctx context.Context, pgURL string, redisURL string, maxConns int32) (*DataStore, error) {
	redisPool := &redis.Pool{
		MaxActive: int(maxConns),
		MaxIdle:   int(maxConns),
		Wait:      true,
		Dial: func() (redis.Conn, error) {
			redisConn, err := redis.DialURL(redisURL)
			if err != nil {
				return nil, fmt.Errorf("redis connection failed: %w", err)
			}
			return redisConn, nil
		},
		IdleTimeout: 200 * time.Second,
	}

	conf, err := pgxpool.ParseConfig(pgURL)
	if err != nil {
		return nil, fmt.Errorf("pg config parsing failed: %w", err)
	}

	conf.MaxConns = maxConns

	pgPool, err := pgxpool.ConnectConfig(ctx, conf)
	if err != nil {
		return nil, fmt.Errorf("pg connection failed: %w", err)
	}
	return &DataStore{
		pgPool:    pgPool,
		redisPool: redisPool,
	}, nil
}

type DataStore struct {
	pgPool    *pgxpool.Pool
	redisPool *redis.Pool
}

func (store *DataStore) Cleanup(ctx context.Context) error {
	store.pgPool.Close()
	return store.redisPool.Close()
}

func (store *DataStore) IsRateLimited(identifier string, resource string) (bool, error) {
	conn := store.redisPool.Get()
	defer conn.Close()

	key := getRateLimitResourceID(identifier, resource)

	exists, err := redis.Bool(conn.Do(
		"EXISTS", key,
	))
	if err != nil {
		return false, fmt.Errorf("failed to look up ip rate limit: %w", err)
	}
	return exists, nil
}

func (store *DataStore) RateLimit(identifier string, resource string, ms int) error {
	key := getRateLimitResourceID(identifier, resource)
	if ms < 1 {
		return nil
	}
	conn := store.redisPool.Get()
	defer conn.Close()
	_, err := conn.Do("SET", key, ms)
	if err != nil {
		return err
	}
	_, err = conn.Do("PEXPIRE", key, ms)
	return err
}

func (store *DataStore) WriteCategory(ctx context.Context, categoryTag string, categoryName string) error {
	_, err := store.pgPool.Exec(ctx, "INSERT INTO cats (tag, name) VALUES ($1, $2)", categoryTag, categoryName)
	if err != nil {
		return err
	}
	return nil
}

func (store *DataStore) RemoveCategory(ctx context.Context, categoryTag string) (int64, error) {
	var affected int64

	tag, err := store.pgPool.Exec(ctx, "DELETE FROM posts WHERE cat = $1", categoryTag)
	if err != nil {
		return affected, err
	}
	affected = tag.RowsAffected()

	tag, err = store.pgPool.Exec(ctx, "DELETE FROM cats WHERE tag = $1", categoryTag)
	if err != nil {
		return affected, err
	}
	return affected + tag.RowsAffected(), nil
}

func (store *DataStore) GetThreadCount(ctx context.Context, categoryTag string) (int, error) {
	var count int
	err := store.pgPool.QueryRow(
		ctx,
		"SELECT COUNT (*) FROM posts WHERE cat = $1 AND parent = 0",
		categoryTag,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to query thread count on %s, %w", categoryTag, err)
	}
	return count, nil
}

func (store *DataStore) GetCategories(ctx context.Context) ([]*Category, error) {
	rows, err := store.pgPool.Query(
		ctx,
		"SELECT tag, name, description, post_count FROM cats",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query categories: %w", err)
	}
	defer rows.Close()

	var cats []*Category = make([]*Category, 0)
	for rows.Next() {
		var c Category
		err := rows.Scan(&c.Tag, &c.Name, &c.Description, &c.PostCount)
		if err != nil {
			return nil, fmt.Errorf("failed to parse a queried category: %w", err)
		}
		cats = append(cats, &c)
	}
	return cats, nil
}

func (store *DataStore) GetPostByNumber(ctx context.Context, categoryTag string, num int) (*Post, error) {
	row := store.pgPool.QueryRow(
		ctx,
		"SELECT num, cat, content, subject, parent, username, created_at FROM posts WHERE cat = $1 AND num = $2",
		categoryTag,
		num,
	)

	var p Post
	err := row.Scan(&p.Num, &p.Cat, &p.Content, &p.Subject, &p.Parent, &p.Username, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to parse a post by number: %w", err)
	}
	return &p, nil
}

func (store *DataStore) GetThreadView(ctx context.Context, categoryTag string, threadNum int) (*ThreadView, error) {

	category, err := store.GetCategory(ctx, categoryTag)
	if err != nil {
		return nil, err
	}

	replyRows, err := store.pgPool.Query(
		ctx,
		"select num, cat, content, subject, parent, username, created_at FROM posts WHERE cat = $1 AND (num = $2 or parent = $2) ORDER BY NUM ASC;",
		category.Tag,
		threadNum,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query thread: %w", err)
	}
	defer replyRows.Close()

	var posts []*Post = make([]*Post, 0)
	for replyRows.Next() {
		post := &Post{}
		err := replyRows.Scan(&post.Num, &post.Cat, &post.Content, &post.Subject, &post.Parent, &post.Username, &post.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse thread reply: %w", err)
		}
		posts = append(posts, post)
	}
	if len(posts) == 0 {
		return nil, ErrNotFound
	}

	return &ThreadView{
		Category: category,
		Posts:    posts,
	}, nil
}

func (store *DataStore) GetCategory(ctx context.Context, categoryTag string) (*Category, error) {
	rows, err := store.pgPool.Query(
		ctx,
		"SELECT name, description, post_count FROM cats WHERE tag = $1",
		categoryTag,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query a category: %w", err)
	}
	defer rows.Close()

	cat := &Category{
		Tag: categoryTag,
	}
	if rows.Next() {
		rows.Scan(&cat.Name, &cat.Description, &cat.PostCount)
		return cat, nil
	}
	return nil, ErrNotFound
}

func (store *DataStore) GetCategoryView(ctx context.Context, categoryTag string) (*CatView, error) {
	cat, err := store.GetCategory(ctx, categoryTag)
	if err != nil {
		return nil, err
	}

	rows, err := store.pgPool.Query(
		ctx,
		"SELECT num, cat, content, subject, username, created_at FROM posts WHERE cat = $1 AND parent = 0 ORDER BY num ASC",
		categoryTag,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query category threads: %w", err)
	}
	defer rows.Close()

	var posts []*Post = make([]*Post, 0)
	for rows.Next() {
		post := &Post{}
		err := rows.Scan(&post.Num, &post.Cat, &post.Content, &post.Subject, &post.Username, &post.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse a queried category view: %w", err)
		}
		posts = append(posts, post)
	}
	return &CatView{
		Threads:  posts,
		Category: cat,
	}, nil
}

func (store *DataStore) WritePost(
	ctx context.Context,
	categoryTag string,
	parentThreadNumber int,
	subject string,
	content string,
	username string,
	email string,
	ip string,
) error {
	_, err := store.pgPool.Exec(
		ctx,
		"CALL write_post($1, $2::int, $3, $4, $5, $6, $7)",
		categoryTag,
		parentThreadNumber,
		content,
		subject,
		username,
		email,
		ip,
	)

	// Catch foreign-key violations and return a human-readable message.
	// Assumes all FK violations are invalid post categories.
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return ErrNotFound
		}
		return fmt.Errorf("failed to execute post write: %w", err)
	}
	return nil
}

func (store *DataStore) RemovePost(ctx context.Context, categoryTag string, number int) (int, error) {
	res, err := store.pgPool.Exec(ctx, "DELETE FROM posts WHERE cat = $1 AND num = $2", categoryTag, number)
	if err != nil {
		return 0, fmt.Errorf("failed to delete post: %w", err)
	}
	return (int)(res.RowsAffected()), nil

}

func (store *DataStore) Migrate(ctx context.Context, up bool) error {
	var file string
	if up {
		file = "migrate_up.sql"
	} else {
		file = "migrate_down.sql"
	}

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path.Join(wd, "db", file))
	if err != nil {
		return err
	}

	_, err = store.pgPool.Exec(ctx, string(data))
	if err != nil {
		return fmt.Errorf("failed to migrate db: %w", err)
	}
	return nil
}

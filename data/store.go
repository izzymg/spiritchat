package data

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

var ErrNotFound = errors.New("not found")

func getIPLockKey(ip string) string {
	return ip + ":lock"
}

// Category contains JSON information describing a Category for posts.
type Category struct {
	Name string `json:"name"`
}

// Post contains JSON information describing a thread, or reply to a thread.
type Post struct {
	Num       int       `json:"num"`
	Cat       string    `json:"cat"`
	Parent    int       `json:"-"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

// UserPost contains JSON information describing an incoming post for writing.
type UserPost struct {
	Content string `json:"content"`
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
func NewDatastore(ctx context.Context, pgURL string, redisURL string, maxConns int32) (*Store, error) {
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
	return &Store{
		pgPool:    pgPool,
		redisPool: redisPool,
	}, nil
}

// Store allows for writing and reading from the persistent data store.
type Store struct {
	pgPool    *pgxpool.Pool
	redisPool *redis.Pool
}

// Cleanup cleans the underlying connection to the data store.
func (store *Store) Cleanup(ctx context.Context) error {
	store.pgPool.Close()
	return store.redisPool.Close()
}

// IsRateLimited returns true if the given IP is being rate limited.
func (store *Store) IsRateLimited(ip string) (bool, error) {
	conn := store.redisPool.Get()
	defer conn.Close()
	exists, err := redis.Bool(conn.Do(
		"EXISTS", getIPLockKey(ip),
	))
	if err != nil {
		return false, fmt.Errorf("failed to look up ip rate limit: %w", err)
	}
	return exists, nil
}

// RateLimit marks IP as rate limited for n seconds.
func (store *Store) RateLimit(ip string, seconds int) error {
	if seconds < 1 {
		return nil
	}
	conn := store.redisPool.Get()
	defer conn.Close()
	_, err := conn.Do("SET", getIPLockKey(ip), seconds)
	if err != nil {
		return err
	}
	_, err = conn.Do("EXPIRE", getIPLockKey(ip), seconds)
	return err
}

// WriteCategory adds a new category to the database.
func (store *Store) WriteCategory(ctx context.Context, catName string) error {
	_, err := store.pgPool.Exec(ctx, "INSERT INTO cats (name) VALUES ($1)", catName)
	if err != nil {
		return err
	}
	return nil
}

/*
RemoveCategory removes ALL posts under category catName and removes the category.
Returns the number of rows affected (1 + number of removed posts).
*/
func (store *Store) RemoveCategory(ctx context.Context, catName string) (int64, error) {
	var affected int64

	tag, err := store.pgPool.Exec(ctx, "DELETE FROM posts WHERE cat = $1", catName)
	if err != nil {
		return affected, err
	}
	affected = tag.RowsAffected()

	tag, err = store.pgPool.Exec(ctx, "DELETE FROM cats WHERE name = $1", catName)
	if err != nil {
		return affected, err
	}
	return affected + tag.RowsAffected(), nil
}

// GetThreadCount returns the number of threads in a category.
func (store *Store) GetThreadCount(ctx context.Context, catName string) (int, error) {
	var count int
	err := store.pgPool.QueryRow(
		ctx,
		"SELECT COUNT (*) FROM posts WHERE cat = $1 AND parent = 0",
		catName,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to query thread count on %s, %w", catName, err)
	}
	return count, nil
}

// GetCategories returns all categories.
func (store *Store) GetCategories(ctx context.Context) ([]*Category, error) {
	rows, err := store.pgPool.Query(
		ctx,
		"SELECT name FROM cats",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query categories: %w", err)
	}
	defer rows.Close()

	var cats []*Category
	for rows.Next() {
		var c Category
		err := rows.Scan(&c.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to parse a queried category: %w", err)
		}
		cats = append(cats, &c)
	}
	return cats, nil
}

/*
GetPostByNumber returns a post in a category by its number.
Will return ErrNotFound if no such post.
*/
func (store *Store) GetPostByNumber(ctx context.Context, catName string, num int) (*Post, error) {
	row := store.pgPool.QueryRow(
		ctx,
		"SELECT num, cat, content, parent, created_at FROM posts WHERE cat = $1 AND num = $2",
		catName,
		num,
	)

	var p Post
	err := row.Scan(&p.Num, &p.Cat, &p.Content, &p.Parent, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to parse a post by number: %w", err)
	}
	return &p, nil
}

/*
GetThreadView returns all the posts in a thread, and the category they're on.
May return ErrNotFound if the requested thread is not an OP thread, or the category
is invalid
*/
func (store *Store) GetThreadView(ctx context.Context, catName string, threadNum int) (*ThreadView, error) {

	replyRows, err := store.pgPool.Query(
		ctx,
		"select num, cat, content, parent, created_at FROM posts WHERE cat = $1 AND (num = $2 or parent = $2) ORDER BY NUM ASC;",
		catName,
		threadNum,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query thread: %w", err)
	}
	defer replyRows.Close()

	posts := []*Post{}
	for replyRows.Next() {
		post := &Post{}
		err := replyRows.Scan(&post.Num, &post.Cat, &post.Content, &post.Parent, &post.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse thread reply: %w", err)
		}
		posts = append(posts, post)
	}
	if len(posts) == 0 {
		return nil, ErrNotFound
	}

	return &ThreadView{
		Category: &Category{
			Name: catName,
		},
		Posts: posts,
	}, nil
}

/*
GetCategory returns a single category. May return ErrNotFound if the given category
name is invalid.
*/
func (store *Store) GetCategory(ctx context.Context, catName string) (*Category, error) {
	rows, err := store.pgPool.Query(
		ctx,
		"SELECT name FROM cats WHERE name = $1",
		catName,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query a category: %w", err)
	}
	defer rows.Close()

	var cat Category
	if rows.Next() {
		rows.Scan(&cat.Name)
		return &cat, nil
	}
	return nil, ErrNotFound
}

/*
GetCatView returns information about a category, and all the threads on it.
May return an ErrNotFound if the given category name is invalid.
*/
func (store *Store) GetCatView(ctx context.Context, catName string) (*CatView, error) {
	cat, err := store.GetCategory(ctx, catName)
	if err != nil {
		return nil, err
	}

	rows, err := store.pgPool.Query(
		ctx,
		"SELECT num, cat, content, created_at FROM posts WHERE cat = $1 AND parent = 0 ORDER BY num ASC",
		catName,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query category threads: %w", err)
	}
	defer rows.Close()

	var posts []*Post
	for rows.Next() {
		post := &Post{}
		err := rows.Scan(&post.Num, &post.Cat, &post.Content, &post.CreatedAt)
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

/*
Writes a post to the database attached to the given category.
Optional parent thread can be provided if it's a reply.
ErrNotFound if invalid post or category.
*/
func (store *Store) WritePost(ctx context.Context, catName string, parentThreadNumber int, p *UserPost) error {
	_, err := store.pgPool.Exec(
		ctx,
		"CALL write_post($1, $2::int, $3)",
		catName,
		parentThreadNumber,
		p.Content,
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

func (store *Store) Migrate(ctx context.Context, up bool) error {
	var file string
	if up {
		file = "./db/migrate_up.sql"
	} else {
		file = "./db/migrate_down.sql"
	}

	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	_, err = store.pgPool.Exec(ctx, string(data))
	if err != nil {
		return fmt.Errorf("failed to migrate db: %w", err)
	}
	return nil
}

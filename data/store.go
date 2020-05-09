package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/rs/xid"
)

// FKViolation is the SQL State error code for foreign-key violations.
const fkViolation = "23503"

const maxContentLen = 300

const minContentLen = 2

// InvalidContentLen is a message describing an invalid post content length.
var InvalidContentLen = fmt.Sprintf(
	"Content must be between %d and %d characters",
	minContentLen,
	maxContentLen,
)

// ErrNotFound is a generic user-friendly not found message.
var ErrNotFound = errors.New("That category or post does not exist")

// Category contains JSON information describing a Category for posts.
type Category struct {
	Name string `json:"name"`
}

// Post contains JSON information describing a thread, or reply to a thread.
type Post struct {
	UID       string    `json:"-"`
	Num       int       `json:"num"`
	Cat       string    `json:"cat"`
	ParentUID string    `json:"-"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

// UserPost contains JSON information describing an incoming post for writing.
type UserPost struct {
	Content string `json:"content"`
}

// IsReply returns true if this post has a parent.
func (post Post) IsReply() bool {
	return len(post.ParentUID) > 0
}

// CatView contains JSON information about a category, and all the threads on it.
type CatView struct {
	Category *Category `json:"category"`
	Threads  []*Post   `json:"threads"`
}

/*
ThreadView contains JSON information about all
the posts in a thread, and the category its on. */
type ThreadView struct {
	Category *Category `json:"category"`
	Posts    []*Post   `json:"posts"`
}

// NewDatastore creates a new data store, creating a connection.
func NewDatastore(ctx context.Context, url string) (*Store, error) {
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("db connection failed: %w", err)
	}
	return &Store{
		connection: conn,
	}, nil
}

// Store allows for writing and reading from the persistent data store.
type Store struct {
	connection *pgx.Conn
}

// Cleanup cleans the underlying connection to the data store.
func (store *Store) Cleanup(ctx context.Context) error {
	return store.connection.Close(ctx)
}

// GetCategories returns all categories.
func (store *Store) GetCategories(ctx context.Context) ([]*Category, error) {
	rows, err := store.connection.Query(
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
Will return ErrNotFound if no such post. */
func (store *Store) GetPostByNumber(ctx context.Context, catName string, num int) (*Post, error) {
	row := store.connection.QueryRow(
		ctx,
		"SELECT uid, num, cat, content, parent, created_at FROM posts WHERE cat = $1 AND num = $2",
		catName,
		num,
	)

	var p Post
	var parent sql.NullString
	err := row.Scan(&p.UID, &p.Num, &p.Cat, &p.Content, &parent, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to parse a post by number: %w", err)
	}
	p.ParentUID = parent.String
	return &p, nil
}

/*
GetThreadView returns all the posts in a thread, and the category they're on.
May return ErrNotFound if the requested thread is not an OP thread, or the category
is invalid */
func (store *Store) GetThreadView(ctx context.Context, catName string, threadNum int) (*ThreadView, error) {

	// Find the category, ensure it's valid
	category, err := store.GetCategory(ctx, catName)
	if err != nil {
		return nil, err
	}

	// Find the OP, ensure it's valid
	op, err := store.GetPostByNumber(ctx, catName, threadNum)
	if err != nil {
		return nil, err
	}
	if op.IsReply() {
		return nil, ErrNotFound
	}

	replyRows, err := store.connection.Query(
		ctx,
		"SELECT uid, num, cat, content, parent, created_at FROM posts WHERE parent = $1 ORDER BY num ASC",
		op.UID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query thread replies: %w", err)
	}
	defer replyRows.Close()

	// Append all the replies after the OP
	posts := []*Post{op}
	for replyRows.Next() {
		post := &Post{}
		var parent sql.NullString
		err := replyRows.Scan(&post.UID, &post.Num, &post.Cat, &post.Content, &parent, &post.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse thread reply: %w", err)
		}
		post.ParentUID = parent.String
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

/*
GetCategory returns a single category. May return ErrNotFound if the given category
name is invalid. */
func (store *Store) GetCategory(ctx context.Context, catName string) (*Category, error) {
	rows, err := store.connection.Query(
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
May return an ErrNotFound if the given category name is invalid. */
func (store *Store) GetCatView(ctx context.Context, catName string) (*CatView, error) {
	cat, err := store.GetCategory(ctx, catName)
	if err != nil {
		return nil, err
	}

	rows, err := store.connection.Query(
		ctx,
		"SELECT uid, num, cat, content, created_at FROM posts WHERE cat = $1 AND parent IS NULL ORDER BY num ASC",
		catName,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query category threads: %w", err)
	}
	defer rows.Close()

	var posts []*Post
	for rows.Next() {
		post := &Post{}
		err := rows.Scan(&post.UID, &post.Num, &post.Cat, &post.Content, &post.CreatedAt)
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
WritePost will record the writing of a post onto the transaction.
Generates a unique ID for the post, and saves only its category, parent
and content. May throw ErrNotFound. */
func (store *Store) WritePost(ctx context.Context, catName string, threadNum int, p *UserPost) error {

	opUID := ""
	if threadNum != 0 {
		op, err := store.GetPostByNumber(ctx, catName, threadNum)
		if err != nil || op.IsReply() {
			return ErrNotFound
		}
		opUID = op.UID
	}

	tx, err := store.connection.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to obtain tx for post write: %w", tx)
	}
	defer tx.Rollback(ctx)

	// Write post procedure expects OP UID

	_, err = tx.Exec(
		ctx,
		"CALL write_post($1, $2, $3, $4)",
		generateUniqueID(),
		catName,
		opUID,
		p.Content,
	)

	// Catch foreign-key violations and return a human-readable message.
	// Assumes all FK violations are invalid post categories.
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == fkViolation {
			return ErrNotFound
		}
		return fmt.Errorf("failed to execute post write: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit post write: %w", err)
	}
	return nil
}

// GenerateUniqueID generates a new globally unique identifier string.
func generateUniqueID() string {
	return xid.New().String()
}

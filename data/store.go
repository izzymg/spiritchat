package data

import (
	"context"
	"errors"
	"fmt"
	"html"
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

// ErrInvalidCategory describes a human readable error for an invalid category.
var ErrInvalidCategory = errors.New("That category does not exist")

// Category contains JSON information describing a Category for posts.
type Category struct {
	Name string `json:"name"`
}

// Post contains JSON information describing a thread, or reply to a thread.
type Post struct {
	UID       string    `json:"-"`
	Num       int       `json:"num"`
	Cat       string    `json:"cat"`
	Parent    string    `json:"parent"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

// CatView contains JSON information about a category, and all the threads on it.
type CatView struct {
	Category
	Threads []Post `json:"threads"`
}

/*
CheckContent validates a post's contents, returning the content sanitized as
the first argument, or a human-readable error message as the second. */
func CheckContent(content string) (string, string) {
	content = html.EscapeString(content)
	if len(content) < minContentLen || len(content) > maxContentLen {
		return "", InvalidContentLen
	}
	return content, ""
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
func (store *Store) GetCategories(ctx context.Context) ([]Category, error) {
	rows, err := store.connection.Query(
		ctx,
		"SELECT name FROM cats",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query categories: %w", err)
	}
	defer rows.Close()

	var cats []Category
	for rows.Next() {
		var c Category
		err := rows.Scan(&c.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to parse a queried category: %w", err)
		}
		cats = append(cats, c)
	}
	return cats, nil
}

// GetThread returns all posts in a thread including the OP.
func (store *Store) GetThread(ctx context.Context, threadUID string) ([]Post, error) {
	rows, err := store.connection.Query(
		ctx,
		"SELECT uid, num, cat, content, created_at FROM posts WHERE uid = $1 OR parent = $1 ORDER BY num ASC",
		threadUID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query thread: %w", err)
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var p Post
		err := rows.Scan(&p.UID, &p.Num, &p.Cat, &p.Content, &p.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse a queried category: %w", err)
		}
		posts = append(posts, p)
	}
	return posts, nil
}

// GetCategory returns a single category.
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
	rows.Scan(&cat.Name)
	return &cat, nil
}

// GetCatView returns information about a category, and all the threads on it.
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

	var posts []Post
	for rows.Next() {
		var p Post
		err := rows.Scan(&p.UID, &p.Num, &p.Cat, &p.Content, &p.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse a queried category view: %w", err)
		}
		posts = append(posts, p)
	}

	return &CatView{
		Threads:  posts,
		Category: *cat,
	}, nil
}

// GetPosts returns all posts in a category.
func (store *Store) GetPosts(ctx context.Context, cat string) ([]Post, error) {
	rows, err := store.connection.Query(
		ctx,
		"SELECT uid, parent, content FROM posts WHERE cat = $1",
		cat,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query posts: %w", err)
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		p := Post{
			Cat: cat,
		}
		err := rows.Scan(&p.UID, &p.Parent, &p.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to parse a queried post: %w", err)
		}
		posts = append(posts, p)
	}
	return posts, nil
}

// Trans creates a new data store transaction, for write operations to the store.
func (store *Store) trans(ctx context.Context) (*Trans, error) {
	tx, err := store.connection.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start db transaction: %w", err)
	}
	return &Trans{tx}, nil
}

// Trans represents a transaction within the data store.
type Trans struct {
	tx pgx.Tx
}

// Commit will push all write operations recorded to the data store.
func (t *Trans) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

// WritePost will record the writing of a post onto the transaction.
func (t *Trans) WritePost(ctx context.Context, p *Post) error {

	_, err := t.tx.Exec(
		ctx,
		"CALL write_post($1, $2, $3, $4)",
		p.UID,
		p.Cat,
		p.Parent,
		p.Content,
	)

	// Catch foreign-key violations and return a human-readable message.
	// Assumes all FK violations are invalid post categories.
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == fkViolation {
			return ErrInvalidCategory
		}
		return fmt.Errorf("failed to execute post write: %w", err)
	}
	return nil
}

// GenerateUniqueID generates a new globally unique identifier string.
func generateUniqueID() string {
	return xid.New().String()
}

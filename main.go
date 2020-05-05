package main

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

const connectionURL = "postgres://postgres:ferret@localhost:5432/spiritchat"

// FKViolation is the SQL State error code for foreign-key violations.
const fkViolation = "23503"

const maxContentLen = 300

const minContentLen = 2

var invalidContentLen = fmt.Sprintf(
	"Content must be between %d and %d characters",
	minContentLen,
	maxContentLen,
)

// ErrInvalidCat describes a human readable error for an invalid category.
var errInvalidCat = errors.New("That category does not exist")

// Category contains JSON information describing a category for posts.
type category struct {
	Name string `json:"name"`
}

// Post contains JSON information describing a thread, or reply to a thread.
type post struct {
	UID       string    `json:"-"`
	Num       int       `json:"num"`
	Cat       string    `json:"cat"`
	Parent    string    `json:"parent"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

// CatView contains JSON information about a category, and all the threads on it.
type catview struct {
	category
	Threads []post `json:"threads"`
}

/*
CheckContent validates a post's contents, returning the content sanitized as
the first argument, or a human-readable error message as the second. */
func checkContent(content string) (string, string) {
	content = html.EscapeString(content)
	if len(content) < minContentLen || len(content) > maxContentLen {
		return "", invalidContentLen
	}
	return content, ""
}

// NewDatastore creates a new data store, creating a connection.
func newDatastore(ctx context.Context, url string) (*datastore, error) {
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("db connection failed: %w", err)
	}
	return &datastore{
		connection: conn,
	}, nil
}

// Datastore allows for writing and reading from the persistent data store.
type datastore struct {
	connection *pgx.Conn
}

// Cleanup cleans the underlying connection to the data store.
func (store *datastore) cleanup(ctx context.Context) error {
	return store.connection.Close(ctx)
}

// GetCatagories returns all categories.
func (store *datastore) getCatagories(ctx context.Context) ([]category, error) {
	rows, err := store.connection.Query(
		ctx,
		"SELECT name FROM cats",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query categories: %w", err)
	}
	defer rows.Close()

	var cats []category
	for rows.Next() {
		var c category
		err := rows.Scan(&c.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to parse a queried category: %w", err)
		}
		cats = append(cats, c)
	}
	return cats, nil
}

// GetThread returns all posts in a thread including the OP.
func (store *datastore) getThread(ctx context.Context, threadUID string) ([]post, error) {
	rows, err := store.connection.Query(
		ctx,
		"SELECT uid, num, cat, content, created_at FROM posts WHERE uid = $1 OR parent = $1 ORDER BY num ASC",
		threadUID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query thread: %w", err)
	}
	defer rows.Close()

	var posts []post
	for rows.Next() {
		var p post
		err := rows.Scan(&p.UID, &p.Num, &p.Cat, &p.Content, &p.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse a queried category: %w", err)
		}
		posts = append(posts, p)
	}
	return posts, nil
}

// GetCatagory returns a single category.
func (store *datastore) getCategory(ctx context.Context, catName string) (*category, error) {
	rows, err := store.connection.Query(
		ctx,
		"SELECT name FROM cats WHERE name = $1",
		catName,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query a category: %w", err)
	}
	defer rows.Close()

	var cat category
	rows.Scan(&cat.Name)
	return &cat, nil
}

// GetCatView returns information about a category, and all the threads on it.
func (store *datastore) getCatView(ctx context.Context, catName string) (*catview, error) {
	cat, err := store.getCategory(ctx, catName)
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

	var posts []post
	for rows.Next() {
		var p post
		err := rows.Scan(&p.UID, &p.Num, &p.Cat, &p.Content, &p.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse a queried category view: %w", err)
		}
		posts = append(posts, p)
	}

	return &catview{
		Threads:  posts,
		category: *cat,
	}, nil
}

// GetPosts returns all posts in a category.
func (store *datastore) getPosts(ctx context.Context, cat string) ([]post, error) {
	rows, err := store.connection.Query(
		ctx,
		"SELECT uid, parent, content FROM posts WHERE cat = $1",
		cat,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query posts: %w", err)
	}
	defer rows.Close()

	var posts []post
	for rows.Next() {
		p := post{
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
func (store *datastore) trans(ctx context.Context) (*trans, error) {
	tx, err := store.connection.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start db transaction: %w", err)
	}
	return &trans{tx}, nil
}

// Trans represents a transaction within the data store.
type trans struct {
	tx pgx.Tx
}

// Commit will push all write operations recorded to the data store.
func (t *trans) commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

// WritePost will record the writing of a post onto the transaction.
func (t *trans) writePost(ctx context.Context, p *post) error {

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
			return errInvalidCat
		}
		return fmt.Errorf("failed to execute post write: %w", err)
	}
	return nil
}

// GenerateUniqueID generates a new globally unique identifier string.
func generateUniqueID() string {
	return xid.New().String()
}

func main() {
}

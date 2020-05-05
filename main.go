package main

import (
	"context"
	"errors"
	"fmt"
	"html"

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
	UID     string
	Cat     string
	Parent  string
	Content string
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
		"INSERT INTO posts (uid, cat, parent, content) VALUES ($1, $2, $3, $4)",
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

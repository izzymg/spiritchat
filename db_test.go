package main

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// Test distinguishing invalid category errors on writes.
func TestInvalidPostCategory(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store, err := newDatastore(ctx, connectionURL)
	if err != nil {
		t.Fatal(fmt.Errorf("failed to create datastore: %w", err))
	}
	defer store.cleanup(context.Background())

	ctx, cancel = context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	tr, err := store.trans(ctx)
	if err != nil {
		t.Fatal(fmt.Errorf("failed to begin transaction: %w", err))
	}
	err = tr.writePost(ctx, &post{
		Cat:     "; DROP TABLE posts",
		Content: "hello!!!",
		UID:     generateUniqueID(),
	})
	if err != nil {
		if !errors.Is(err, errInvalidCat) {
			t.Fatal(fmt.Errorf("failed to write post: %w", err))
		}
		t.Log(err)
	}
}

func genStr(n int, s string) string {
	if n == 1 {
		return s
	}
	return genStr(n-1, s+"a")
}

// Test sanitizing a post's content.
func TestCheckContent(t *testing.T) {
	onMin := genStr(minContentLen, "a")
	belowMin := genStr(minContentLen-1, "a")
	onMax := genStr(maxContentLen, "a")
	aboveMax := genStr(maxContentLen+1, "a")

	_, errStr := checkContent(onMin)
	if len(errStr) > 0 {
		t.Fatal("Expected no err string")
	}

	_, errStr = checkContent(belowMin)
	if len(errStr) == 0 {
		t.Fatal("Expected an err string")
	}

	_, errStr = checkContent(onMax)
	if len(errStr) > 0 {
		t.Fatal("Expected no err string")
	}

	_, errStr = checkContent(aboveMax)
	if len(errStr) == 0 {
		t.Fatal("Expected an err string")
	}
}

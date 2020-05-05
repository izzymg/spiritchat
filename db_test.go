package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"testing"
)

// Test distinguishing invalid category errors on writes.
func TestInvalidPostCategory(t *testing.T) {
	ctx := context.Background()
	store, err := newDatastore(ctx, connectionURL)
	if err != nil {
		t.Fatal(fmt.Errorf("failed to create datastore: %w", err))
	}
	defer store.cleanup(ctx)

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

func TestGetThread(t *testing.T) {
	ctx := context.Background()
	store, err := newDatastore(ctx, connectionURL)
	if err != nil {
		t.Fatal(fmt.Errorf("failed to create datatstore: %w", err))
	}

	defer store.cleanup(ctx)

	thread, err := store.getThread(ctx, "op4")
	if err != nil {
		t.Fatal(err)
	}

	log.Println(thread)
}

func TestGetCatView(t *testing.T) {
	ctx := context.Background()
	store, err := newDatastore(ctx, connectionURL)
	if err != nil {
		t.Fatal(fmt.Errorf("failed to create datatstore: %w", err))
	}

	defer store.cleanup(ctx)

	catView, err := store.getCatView(ctx, "animals")
	if err != nil {
		t.Fatal(err)
	}

	log.Println(catView)
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

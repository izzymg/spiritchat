package data

import (
	"context"
	"errors"
	"spiritchat/config"
	"sync"
	"testing"
	"time"
)

// Should return true if a post is a reply in the DB.
func TestIsReply(t *testing.T) {
	thread := Post{
		Parent: 0,
	}
	replyOne := Post{
		Parent: 1,
	}

	replyTwo := Post{
		Parent: 300,
	}

	if thread.IsReply() {
		t.Error("thread should not be reply")
	}

	if !replyOne.IsReply() {
		t.Error("reply should be reply")
	}

	if !replyTwo.IsReply() {
		t.Error("reply should be reply")
	}
}

func TestIntegrations(t *testing.T) {
	shouldRun, store, err := getIntegrationTestSetup()
	if err != nil {
		t.Fatalf("integration test setup failure: %v", err)
	}
	if !shouldRun {
		t.Log("skipping integration test")
		return
	}

	ctx := context.Background()
	defer store.Cleanup(ctx)

	integrationTests := map[string]func(context.Context, *Store) func(t *testing.T){
		"Post writes":        integration_WritePosts,
		"Get Category View":  integration_GetCatView,
		"Get Categories":     integration_GetCategories,
		"Get Post by Number": integration_GetPostByNumber,
		"Get Thread View":    integration_GetThreadView,
		"Rate limit IPs":     integration_RateLimit,
	}

	for name, fn := range integrationTests {
		t.Run(name, fn(ctx, store))
	}

	t.Run("Test Concurrent Thread Writes", integration_ConcurrentThreadWrites(ctx, store))

}

// Returns whether integrations should run, and the given store if so.
func getIntegrationTestSetup() (bool, *Store, error) {
	conf, shouldRun := config.GetIntegrationsConfig()
	if !shouldRun {
		return false, nil, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store, err := NewDatastore(ctx, conf.PGURL, conf.RedisURL, 100)
	if err != nil {
		return true, nil, err
	}
	return true, store, nil
}

func integration_GetThreadView(ctx context.Context, store *Store) func(t *testing.T) {
	return func(t *testing.T) {
		_, err := store.GetThreadView(ctx, "none", 0)
		if err == nil || err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}

		testCategories := []string{"bbb", "vvv", "ccc"}
		tests := map[string]int{
			testCategories[0]: 5,
			testCategories[1]: 15,
			testCategories[2]: 0,
		}

		err = createTestCategories(ctx, store, testCategories)
		if err != nil {
			t.Fatal(err)
		}
		defer removeTestCategories(ctx, store, testCategories)

		// invalid
		_, err = store.GetThreadView(ctx, "nothing", 0)
		if err == nil || err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}

		testPost := createTestUserPost()
		opCount := 3
		for categoryName, replyCount := range tests {
			// create OPs
			for i := 0; i < opCount; i++ {
				err := store.WritePost(ctx, categoryName, 0, testPost)
				if err != nil {
					t.Fatal(err)
				}
			}

			opNum := opCount - 1
			// create replies to an op
			for i := 0; i < replyCount; i++ {
				err := store.WritePost(ctx, categoryName, opNum, testPost)
				if err != nil {
					t.Fatal(err)
				}
			}

			view, err := store.GetThreadView(ctx, categoryName, opNum)
			if err != nil {
				t.Fatal(err)
			}
			if len(view.Posts) != replyCount+1 {
				t.Errorf("expected %d posts, got: %d", replyCount+1, len(view.Posts))
			}
		}
	}
}

func integration_GetPostByNumber(ctx context.Context, store *Store) func(t *testing.T) {
	return func(t *testing.T) {

		testCategories := []string{"beepboop", "bonk"}
		err := createTestCategories(ctx, store, testCategories)
		if err != nil {
			t.Error(err)
		}
		defer removeTestCategories(ctx, store, testCategories)

		testPost := createTestUserPost()
		for _, categoryName := range testCategories {
			err = store.WritePost(ctx, categoryName, 0, testPost)
			if err != nil {
				t.Error(err)
			}
			post, err := store.GetPostByNumber(ctx, categoryName, 1)
			if err != nil {
				t.Error(err)
			}

			if post.Content != testPost.Content {
				t.Errorf("post content mismatch, expected %s got: %s", testPost.Content, post.Content)
			}
		}

		// test invalid post
		_, err = store.GetPostByNumber(ctx, "i dont exist", 0)
		if err == nil || !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	}
}

func integration_GetCategories(ctx context.Context, store *Store) func(t *testing.T) {
	return func(t *testing.T) {
		tests := map[string][]string{
			"Some categories": {"beep", "boop", "bop"},
			"No categories":   {},
		}

		for name, categoryNames := range tests {
			t.Run(name, func(t *testing.T) {
				err := createTestCategories(ctx, store, categoryNames)
				if err != nil {
					t.Error(err)
				}
				defer removeTestCategories(ctx, store, categoryNames)

				cats, err := store.GetCategories(ctx)
				if err != nil {
					t.Error(err)
				}
				if len(cats) != len(categoryNames) {
					t.Errorf("expected %d categories, got: %d %v", len(categoryNames), len(cats), cats)
				}
				for i := 0; i < len(categoryNames); i++ {
					has := false
					for j := 0; j < len(cats); j++ {
						if cats[j].Name == categoryNames[i] {
							has = true
						}
					}
					if !has {
						t.Errorf("returned categories does not have value: %s", categoryNames[i])
					}
				}
			})
		}
	}
}

func integration_GetCatView(ctx context.Context, store *Store) func(t *testing.T) {
	return func(t *testing.T) {

		testCategories := []string{"test-catview"}
		threadCount := 5

		// store a category
		err := createTestCategories(ctx, store, testCategories)
		if err != nil {
			t.Error(err)
		}
		defer removeTestCategories(ctx, store, testCategories)

		// write a thread into the category
		for i := 0; i < threadCount; i++ {
			err = store.WritePost(ctx, testCategories[0], 0, createTestUserPost())
			if err != nil {
				t.Error(err)
			}
		}

		// write a reply to that post
		err = store.WritePost(ctx, testCategories[0], 1, createTestUserPost())
		if err != nil {
			t.Error(err)
		}

		// getcatview should return the category, the post, but no replies
		view, err := store.GetCatView(ctx, testCategories[0])
		if err != nil {
			t.Error(err)
		}
		if view == nil || view.Category == nil {
			t.Error("got nil category")
		}
		if len(view.Threads) != threadCount {
			t.Errorf("expected %d threads, got %d", threadCount, len(view.Threads))
		}
		if view.Category.Name != testCategories[0] {
			t.Errorf("expected category name %s, got %s: ", testCategories[0], view.Category.Name)
		}
	}
}

func integration_ConcurrentThreadWrites(ctx context.Context, store *Store) func(t *testing.T) {
	return func(t *testing.T) {
		categoryThreadCountMap := map[string]int{
			"test-1": 45,
			"test-2": 22,
			"test-3": 10,
		}
		categoryNames := []string{"test-1", "test-2", "test-3"}

		err := createTestCategories(ctx, store, categoryNames)
		if err != nil {
			t.Error(err)
		}
		defer removeTestCategories(ctx, store, categoryNames)

		t.Run("Concurent thread writes", concurrentThreadWriteTest(ctx, store, categoryThreadCountMap))
	}
}

/*
*
Test writing valid & invalid posts
*/
func integration_WritePosts(ctx context.Context, datastore *Store) func(t *testing.T) {
	return func(t *testing.T) {
		t.Run("invalid category", func(t *testing.T) {
			err := datastore.WritePost(ctx, "invalid-category", 0, createTestUserPost())
			if err == nil {
				t.Errorf("expected writepost error, got: %v", err)
			}
			if !errors.Is(err, ErrNotFound) {
				t.Errorf("expected an ErrNotFound from writepost, got: %v", err)
			}
		})

		t.Run("valid category, valid thread", func(t *testing.T) {
			testCategories := []string{"test-cat"}
			err := createTestCategories(ctx, datastore, testCategories)
			if err != nil {
				t.Error(err)
			}
			defer removeTestCategories(ctx, datastore, testCategories)

			err = datastore.WritePost(ctx, testCategories[0], 0, createTestUserPost())
			if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})

		t.Run("valid category, invalid parent post", func(t *testing.T) {
			testCategories := []string{"test-cat"}
			createTestCategories(ctx, datastore, testCategories)
			defer removeTestCategories(ctx, datastore, testCategories)

			err := datastore.WritePost(ctx, testCategories[0], 5, createTestUserPost())
			if err == nil || !errors.Is(err, ErrNotFound) {
				t.Errorf("expected ErrNotFound, got: %v", err)
			}
		})
	}
}

// Creates an empty test user post.
func createTestUserPost() *UserPost {
	return &UserPost{
		Content: "test",
	}
}

func createTestCategories(ctx context.Context, datastore *Store, categoryNames []string) error {
	for _, categoryName := range categoryNames {
		err := datastore.WriteCategory(ctx, categoryName)
		if err != nil {
			return err
		}
	}
	return nil
}

func removeTestCategories(ctx context.Context, datastore *Store, categoryNames []string) error {
	for _, categoryName := range categoryNames {
		_, err := datastore.RemoveCategory(ctx, categoryName)
		if err != nil {
			return err
		}
	}
	return nil
}

/*
Takes a map of category names and their number of threads to create.
Creates all categories, and then writes n threads to each category concurrently.
*/
func concurrentThreadWriteTest(ctx context.Context, datastore *Store, tests map[string]int) func(t *testing.T) {
	return func(t *testing.T) {
		for categoryName, threadCount := range tests {
			testUserPost := createTestUserPost()
			threadCount := threadCount
			categoryName := categoryName
			t.Run(categoryName, func(t *testing.T) {
				t.Parallel()
				// write n posts concurrently to a category
				var wg sync.WaitGroup
				categoryName := categoryName
				for i := 0; i < threadCount; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						err := datastore.WritePost(ctx, categoryName, 0, testUserPost)
						if err != nil {
							panic(err)
						}
					}()
				}
				wg.Wait()

				count, err := datastore.GetThreadCount(ctx, categoryName)
				if err != nil {
					t.Fatalf("failed to get thread count on category %s: %v", categoryName, err)
				}
				if count != threadCount {
					t.Errorf("expected %d threads, got %d", threadCount, count)
				}
			})
		}
	}
}

func integration_RateLimit(ctx context.Context, store *Store) func(t *testing.T) {
	return func(t *testing.T) {

		tests := []string{"13.3.4", "100.3r45.5434z", "localhost", "127.0.0.1", "123.123.123.123", "not even an ip lol"}
		limited, err := store.IsRateLimited("garbage_ip")
		if err != nil {
			t.Error(err)
		}
		if limited {
			t.Error("Expected no rate limit on garbage IP")
		}

		timeMs := 50
		for _, ip := range tests {
			store.RateLimit(ip, timeMs)
			limited, err = store.IsRateLimited(ip)
			if err != nil {
				t.Error(err)
			}
			if !limited {
				t.Error("Expected rate limit after limiting")
			}
		}

		<-time.After(time.Duration(timeMs+50) * time.Millisecond)

		for _, ip := range tests {
			limited, err = store.IsRateLimited(ip)
			if err != nil {
				t.Error(err)
			}
			if limited {
				t.Error("Expected rate limit to expire")
			}
		}
	}
}

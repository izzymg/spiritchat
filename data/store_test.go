package data

import (
	"context"
	"errors"
	"spiritchat/config"
	"sync"
	"testing"
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
		"Concurrent Thread Writes": integration_ConcurrentThreadWrites,
		"Post writes":              integration_WritePosts,
		"Get Category View":        integration_GetCatView,
		"Get Categories":           integration_GetCategories,
	}

	for name, fn := range integrationTests {
		t.Run(name, fn(ctx, store))
	}

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
					t.Errorf("expected %d categories, got: %d", len(categoryNames), len(cats))
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

/*
func intRateLimit(ctx context.Context, store *Store, conf *config.Config) func(t *testing.T) {
	return func(t *testing.T) {

		tests := []string{"13.3.4", "100.3r45.5434z", "localhost", "127.0.0.1", "zzzz"}
		for _, ip := range tests {
			ip := ip
			t.Run(fmt.Sprintf("rateLimit-%s", ip), func(t *testing.T) {
				t.Parallel()
				// Random key should not be limited
				limited, err := store.IsRateLimited("garbage_ip")
				if err != nil {
					t.Error(err)
				}
				if limited {
					t.Error("Expected limited == false")
				}

				store.RateLimit(ip, 2)
				limited, err = store.IsRateLimited(ip)
				if err != nil {
					t.Error(err)
				}
				if !limited {
					t.Error("Expected limited == true")
				}

				<-time.After(time.Second * 3)
				limited, err = store.IsRateLimited(ip)
				if err != nil {
					t.Error(err)
				}
				if limited {
					t.Error("Expected limited == false")
				}
			})
		}
	}
}
*/

package data

import (
	"context"
	"spiritchat/config"
	"sync"
	"testing"
)

func TestConcurrentThreadWrites(t *testing.T) {
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

	tests := map[string]int{
		"test-1": 45,
		"test-2": 22,
		"test-3": 10,
	}

	createTestCategories(ctx, store, tests)
	defer removeTestCategories(ctx, store, tests)

	t.Run("Concurent thread writes", concurrentThreadWriteTest(ctx, tests, store))
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

// Creates an empty test user post.
func createTestUserPost() *UserPost {
	return &UserPost{
		Content: "test",
	}
}

func createTestCategories(ctx context.Context, datastore *Store, tests map[string]int) error {
	for categoryName := range tests {
		err := datastore.WriteCategory(ctx, categoryName)
		if err != nil {
			return err
		}
	}
	return nil
}

func removeTestCategories(ctx context.Context, datastore *Store, tests map[string]int) error {
	for categoryName := range tests {
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
func concurrentThreadWriteTest(ctx context.Context, tests map[string]int, datastore *Store) func(t *testing.T) {
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

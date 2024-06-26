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

	integrationTests := map[string]func(context.Context, *DataStore) func(t *testing.T){
		"Post writes":        integration_WritePosts,
		"Get Category View":  integration_GetCategoryView,
		"Get Categories":     integration_GetCategories,
		"Get Post by Number": integration_GetPostByNumber,
		"Get Thread View":    integration_GetThreadView,
		"Remove Posts":       integration_RemovePost,
		"Get Posts by Email": integration_GetPostsByEmail,
	}

	for name, fn := range integrationTests {
		t.Run(name, fn(ctx, store))
	}

	t.Run("Test Concurrent Thread Writes", integration_ConcurrentThreadWrites(ctx, store))

}

// Returns whether integrations should run, and the given store if so.
func getIntegrationTestSetup() (bool, *DataStore, error) {
	conf, shouldRun := config.GetIntegrationsConfig()
	if !shouldRun {
		return false, nil, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store, err := NewDatastore(ctx, conf.PGURL, 100)
	if err != nil {
		return true, nil, err
	}
	return true, store, nil
}

func integration_GetThreadView(ctx context.Context, store *DataStore) func(t *testing.T) {
	return func(t *testing.T) {
		_, err := store.GetThreadView(ctx, "none", 0)
		if err == nil || err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}

		testCategories := map[string]string{"bbb": "vvv", "vvv": "ccc", "ccc": "ddd"}
		tests := map[string]int{
			"bbb": 5,
			"vvv": 15,
			"ccc": 0,
		}

		err = createTestCategories(ctx, store, testCategories)
		if err != nil {
			t.Error(err)
		}
		defer removeTestCategories(ctx, store, testCategories)

		// invalid
		_, err = store.GetThreadView(ctx, "nothing", 0)
		if err == nil || err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}

		opCount := 3
		for tag, replyCount := range tests {
			// create OPs
			for i := 0; i < opCount; i++ {
				err := store.WritePost(ctx, tag, 0, "abc", "bdef", "a", "b", "c")
				if err != nil {
					t.Error(err)
				}
			}

			opNum := opCount - 1
			// create replies to an op
			for i := 0; i < replyCount; i++ {
				err := store.WritePost(ctx, tag, opNum, "abc", "bdef", "a", "b", "c")
				if err != nil {
					t.Error(err)
				}
			}

			view, err := store.GetThreadView(ctx, tag, opNum)
			if err != nil {
				t.Error(err)
			}
			if len(view.Posts) != replyCount+1 {
				t.Errorf("expected %d posts, got: %d", replyCount+1, len(view.Posts))
			}
		}
	}
}

func integration_RemovePost(ctx context.Context, store *DataStore) func(t *testing.T) {
	return func(t *testing.T) {
		testCategories := map[string]string{
			"beep": "boop",
			"bonk": "fonk",
		}

		err := createTestCategories(ctx, store, testCategories)
		if err != nil {
			t.Error(err)
		}
		defer removeTestCategories(ctx, store, testCategories)

		// write parent
		err = store.WritePost(ctx, "beep", 0, "subject", "content", "username", "email", "ip")
		if err != nil {
			t.Error(err)
		}

		// write unrelated parent
		expectSubject := "UNRELATED POST"
		err = store.WritePost(ctx, "beep", 0, expectSubject, "content", "username", "email", "ip")
		if err != nil {
			t.Error(err)
		}

		// write replies
		replyCount := 20
		for i := 0; i < replyCount; i++ {
			err = store.WritePost(ctx, "beep", 1, "subject", "content", "username", "email", "ip")
			if err != nil {
				t.Error(err)
			}
		}

		removed, err := store.RemovePost(ctx, "beep", 1)
		if err != nil {
			t.Error(err)
		}

		// 1 post should be removed
		if removed != 1 {
			t.Errorf("expected %d removed posts, got %d", 1, removed)
		}

		// but all the replies should be gone
		for i := 0; i < replyCount; i++ {
			post, err := store.GetPostByNumber(ctx, "beep", 1+replyCount)
			if err != ErrNotFound {
				t.Errorf("expected no post, got post %+v", post)
			}
		}
		post, err := store.GetPostByNumber(ctx, "beep", 2)
		if err != nil {
			t.Errorf("expected unrelated post still there, got %v", err)
		}
		if post.Subject != expectSubject {
			t.Errorf("expected %s content, got %s", expectSubject, post.Content)
		}
	}
}

func integration_GetPostByNumber(ctx context.Context, store *DataStore) func(t *testing.T) {
	return func(t *testing.T) {

		testCategories := map[string]string{
			"beep": "boop",
			"bonk": "fonk",
		}
		err := createTestCategories(ctx, store, testCategories)
		if err != nil {
			t.Error(err)
		}
		defer removeTestCategories(ctx, store, testCategories)

		expectContent := "beepboop"
		for tag := range testCategories {
			err = store.WritePost(ctx, tag, 0, "hey", expectContent, "a", "b", "c")
			if err != nil {
				t.Error(err)
			}
			post, err := store.GetPostByNumber(ctx, tag, 1)
			if err != nil {
				t.Error(err)
			}

			if post.Content != expectContent {
				t.Errorf("post content mismatch, expected %s got: %s", expectContent, post.Content)
			}
		}

		// test invalid post
		_, err = store.GetPostByNumber(ctx, "i dont exist", 0)
		if err == nil || !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	}
}

func integration_GetCategories(ctx context.Context, store *DataStore) func(t *testing.T) {
	return func(t *testing.T) {
		tests := map[string]map[string]string{
			"Some categories": {
				"xxxx": "zzzz",
				"aaaa": "bbbb",
				"vvvv": "eeeee",
			},
			"No categories": {},
		}

		for name, categories := range tests {
			t.Run(name, func(t *testing.T) {
				err := createTestCategories(ctx, store, categories)
				if err != nil {
					t.Error(err)
				}
				defer removeTestCategories(ctx, store, categories)

				cats, err := store.GetCategories(ctx)
				if err != nil {
					t.Error(err)
				}
				if len(cats) != len(categories) {
					t.Errorf("expected %d categories, got: %d %v", len(categories), len(cats), cats)
				}
				for i := 0; i < len(cats); i++ {
					has := false

					for tag := range categories {
						if cats[i].Tag == tag {
							has = true
						}
					}
					if !has {
						t.Error("mismatch in returned categories")
					}
				}
			})
		}
	}
}

func integration_GetCategoryView(ctx context.Context, store *DataStore) func(t *testing.T) {
	return func(t *testing.T) {

		catName := "beep"
		testCategories := map[string]string{catName: "best"}
		threadCount := 5

		// store a category
		err := createTestCategories(ctx, store, testCategories)
		if err != nil {
			t.Error(err)
		}
		defer removeTestCategories(ctx, store, testCategories)

		// write a thread into the category
		for i := 0; i < threadCount; i++ {
			err = store.WritePost(ctx, catName, 0, "beep", "boop", "a", "b", "c")
			if err != nil {
				t.Error(err)
			}
		}

		// write a reply to that post
		err = store.WritePost(ctx, catName, 1, "beep", "boop", "a", "b", "c")
		if err != nil {
			t.Error(err)
		}

		// GetCategoryView should return the category, the post, but no replies
		view, err := store.GetCategoryView(ctx, catName)
		if err != nil {
			t.Error(err)
		}
		if view == nil || view.Category == nil {
			t.Error("got nil category")
		}
		if len(view.Threads) != threadCount {
			t.Errorf("expected %d threads, got %d", threadCount, len(view.Threads))
		}
		if view.Category.Tag != catName {
			t.Errorf("expected category tag %s, got %s: ", catName, view.Category.Tag)
		}
	}
}

func integration_GetPostsByEmail(ctx context.Context, store *DataStore) func(t *testing.T) {
	return func(t *testing.T) {
		testCategoryTag := "test-category"
		testCategories := map[string]string{testCategoryTag: "test"}
		expectEmail := "coolemail@example.com"
		expectContent := "beep"
		createTestCategories(ctx, store, testCategories)
		defer removeTestCategories(ctx, store, testCategories)

		postCount := 15
		err := store.WritePost(ctx, testCategoryTag, 0, "subject", "otherContent", "username", "another email", "ip")
		if err != nil {
			t.Error(err)
		}

		for i := 0; i < postCount; i++ {
			err := store.WritePost(ctx, testCategoryTag, 0, "subject", expectContent, "username", expectEmail, "ip")
			if err != nil {
				t.Error(err)
			}
		}
		posts, err := store.GetPostsByEmail(ctx, expectEmail)
		if err != nil {
			t.Error(err)
		}
		if len(posts) != postCount {
			t.Errorf("expected %d posts returned, got %d", postCount, len(posts))
		}
		for _, post := range posts {
			if post.Content != expectContent {
				t.Errorf("got unexpected post content %s", post.Content)
			}
		}
	}
}

func integration_ConcurrentThreadWrites(ctx context.Context, store *DataStore) func(t *testing.T) {
	return func(t *testing.T) {
		categoryThreadCountMap := map[string]int{
			"test-1": 45,
			"test-2": 22,
			"test-3": 10,
		}
		testCategories := map[string]string{"test-1": "aa", "test-2": "bb", "test-3": "cc"}

		err := createTestCategories(ctx, store, testCategories)
		if err != nil {
			t.Error(err)
		}
		defer removeTestCategories(ctx, store, testCategories)

		t.Run("Concurent thread writes", concurrentThreadWriteTest(ctx, store, categoryThreadCountMap))
	}
}

/*
*
Test writing valid & invalid posts
*/
func integration_WritePosts(ctx context.Context, datastore *DataStore) func(t *testing.T) {
	return func(t *testing.T) {
		t.Run("invalid category", func(t *testing.T) {
			err := datastore.WritePost(ctx, "invalid-category", 0, "beep", "boop", "a", "b", "c")
			if err == nil {
				t.Errorf("expected writepost error, got: %v", err)
			}
			if !errors.Is(err, ErrNotFound) {
				t.Errorf("expected an ErrNotFound from writepost, got: %v", err)
			}
		})

		t.Run("valid category, valid thread", func(t *testing.T) {
			name := "BEEW"
			testCategories := map[string]string{name: "meowmeow"}
			err := createTestCategories(ctx, datastore, testCategories)
			if err != nil {
				t.Error(err)
			}
			defer removeTestCategories(ctx, datastore, testCategories)

			err = datastore.WritePost(ctx, name, 0, "beep", "boop", "a", "b", "c")
			if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})

		t.Run("valid category, invalid parent post", func(t *testing.T) {
			name := "BEEW"
			testCategories := map[string]string{name: "meow"}
			createTestCategories(ctx, datastore, testCategories)
			defer removeTestCategories(ctx, datastore, testCategories)

			err := datastore.WritePost(ctx, name, 5, "beep", "boop", "a", "b", "c")
			if err == nil || !errors.Is(err, ErrNotFound) {
				t.Errorf("expected ErrNotFound, got: %v", err)
			}
		})
	}
}

func createTestCategories(ctx context.Context, datastore *DataStore, categorys map[string]string) error {
	for tag, name := range categorys {
		err := datastore.WriteCategory(ctx, tag, name)
		if err != nil {
			return err
		}
	}
	return nil
}

func removeTestCategories(ctx context.Context, datastore *DataStore, tags map[string]string) error {
	for tag := range tags {
		_, err := datastore.RemoveCategory(ctx, tag)
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
func concurrentThreadWriteTest(ctx context.Context, datastore *DataStore, tests map[string]int) func(t *testing.T) {
	return func(t *testing.T) {
		for categoryName, threadCount := range tests {
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
						err := datastore.WritePost(ctx, categoryName, 0, "beep", "boop", "a", "b", "c")
						if err != nil {
							panic(err)
						}
					}()
				}
				wg.Wait()

				count, err := datastore.GetThreadCount(ctx, categoryName)
				if err != nil {
					t.Errorf("failed to get thread count on category %s: %v", categoryName, err)
				}
				if count != threadCount {
					t.Errorf("expected %d threads, got %d", threadCount, count)
				}
			})
		}
	}
}

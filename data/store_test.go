package data

import (
	"context"
	"fmt"
	"spiritchat/config"
	"testing"
	"time"

	"github.com/jackc/pgx/v4"
)

func TestIntegration(t *testing.T) {
	conf, shouldRun := config.GetIntegrationsConfig()
	if !shouldRun {
		t.Log("Skipping integration test")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store, err := NewDatastore(ctx, conf.PGURL, conf.RedisURL, 100)
	if err != nil {
		t.Error(err)
	}
	defer store.Cleanup(context.Background())

	t.Run("getThreadCount", intGetThreadCount(ctx, store, conf))
	t.Run("getThreadCount", intRateLimit(ctx, store, conf))

}

func intGetThreadCount(ctx context.Context, store *Store, conf *config.Config) func(t *testing.T) {
	return func(t *testing.T) {
		catName := "integration_GetThreadCount"

		setup := func(threads int) func() {

			if threads < 1 {
				return func() {}
			}

			// Add a test category
			conn, err := pgx.Connect(ctx, conf.PGURL)
			if err != nil {
				panic(err)
			}

			_, err = conn.Exec(
				ctx,
				"INSERT INTO cats (name) VALUES ($1)",
				catName,
			)
			if err != nil {
				panic(err)
			}

			opUID := generateUniqueID()
			addPost := func(uid string, parent string) {
				_, err := conn.Exec(
					ctx,
					"CALL write_post($1, $2, $3, $4)",
					uid,
					catName,
					parent,
					"test_post",
				)
				if err != nil {
					panic(err)
				}
			}

			// Add threads and replies to a thread
			// Since 1 OP is added already, add - 1
			addPost(opUID, "")
			for i := 0; i < threads-1; i++ {
				addPost(generateUniqueID(), "")
				addPost(generateUniqueID(), opUID)
			}

			return func() {
				_, err := conn.Exec(
					ctx,
					"DELETE FROM posts WHERE cat = $1",
					catName,
				)
				if err != nil {
					panic(err)
				}
				_, err = conn.Exec(
					ctx,
					"DELETE FROM cats WHERE name = $1",
					catName,
				)
				conn.Close(context.Background())
			}
		}

		tests := []int{
			100, 1, 2000, 54, 99, 83, 24, 0,
		}

		for _, n := range tests {
			t.Run(fmt.Sprintf("threadCount-%d", n), func(t *testing.T) {
				t.Parallel()
				teardown := setup(n)
				defer teardown()

				start := time.Now()
				c, err := store.GetThreadCount(ctx, catName)
				t.Logf("getThreadCount in %d ms", time.Since(start).Milliseconds())
				if err != nil {
					t.Error(err)
				}
				if c != n {
					t.Errorf("Expected %d threads, got %d", n, c)
				}
			})
		}
	}
}

func intRateLimit(ctx context.Context, store *Store, conf *config.Config) func(t *testing.T) {
	return func(t *testing.T) {

		tests := []string{"13.3.4", "100.3r45.5434z", "localhost", "127.0.0.1", "zzzz"}
		for _, ip := range tests {
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

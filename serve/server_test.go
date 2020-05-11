package serve

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"spiritchat/config"
	"spiritchat/data"
	"testing"

	"github.com/jackc/pgx/v4"
	"golang.org/x/sync/errgroup"
)

// Returns false if integrations shouldn't be run.
func ShouldRunIntegrations() bool {
	if env, exists := os.LookupEnv(
		"SPIRITTEST_INTEGRATIONS",
	); !exists || env == "FALSE" || env == "0" {
		return false
	}
	return true
}

// Returns false if integrations shouldn't be run, or true, and integration config.
func GetIntegrationsConfig() (*config.Config, bool) {
	if !ShouldRunIntegrations() {
		return nil, false
	}
	pgURL := os.Getenv("SPIRITTEST_PG_URL")
	redisURL := os.Getenv("SPIRITTEST_REDIS_URL")
	addr := os.Getenv("SPIRITTEST_ADDR")
	if len(pgURL) == 0 || len(redisURL) == 0 || len(addr) == 0 {
		panic("SPIRITTEST_PG_URL or SPIRITTEST_REDIS_URL or SPIRITTEST_ADDR empty")
	}

	return &config.Config{
		HTTPAddress: addr,
		PGURL:       pgURL,
		RedisURL:    redisURL,
	}, true
}

// Integrations - run on fake DB
func TestIntegration(t *testing.T) {
	conf, shouldRun := GetIntegrationsConfig()
	if !shouldRun {
		t.Log("Skipping integration test...")
		return
	}

	store, err := data.NewDatastore(context.Background(), conf.PGURL, conf.RedisURL)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Cleanup(context.Background())

	/* Create a raw postgres connection and setup the DB,
	returning a function to tear it down.

	Each test will create its own entries in the database
	to run on, allowing concurrency.
	*/
	setup := func(catName string) func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		conn, err := pgx.Connect(ctx, conf.PGURL)
		if err != nil {
			panic(err)
		}

		tx, err := conn.Begin(ctx)
		if err != nil {
			panic(err)
		}
		defer tx.Rollback(ctx)

		_, err = tx.Exec(
			ctx,
			"INSERT INTO cats (name) VALUES ($1)",
			catName,
		)
		if err != nil {
			panic(err)
		}

		err = tx.Commit(ctx)
		if err != nil {
			panic(err)
		}

		return func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			// Remove all assocaited posts, and then the category.
			_, err = conn.Exec(
				ctx,
				"DELETE FROM posts WHERE cat = $1",
				catName,
			)
			_, err = conn.Exec(
				ctx,
				"DELETE FROM cats WHERE name = $1",
				catName,
			)
			if err != nil {
				panic(err)
			}

			conn.Close(context.Background())
		}
	}

	// Post cooldown is disabled
	server := NewServer(store, ServerOptions{
		Address:             conf.HTTPAddress,
		CorsOriginAllow:     "*",
		PostCooldownSeconds: 0,
	})
	go func() {
		if err := server.Listen(context.Background()); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	// Test POSTing new threads
	t.Run("Write threads", func(t *testing.T) {
		catName := "intgr_writeThreadsTest"
		postContent := "some_test_content!"

		t.Log("Setting up writeThreadsTest")
		teardown := setup(catName)
		defer teardown()
		defer t.Log("Tearing down writeThreadsTest")

		body, err := json.Marshal(data.UserPost{
			Content: postContent,
		})
		if err != nil {
			t.Error(err)
		}

		for i := 0; i < 50; i++ {
			var g errgroup.Group

			g.Go(func() error {
				res, err := http.Post("http://"+conf.HTTPAddress+"/v1/"+catName+"/0", "application/JSON", bytes.NewReader(body))
				if err != nil {
					return fmt.Errorf("writeThread got unexpected http error: %w", err)
				}
				if res.StatusCode != 200 {
					return fmt.Errorf("writeThread expected status code 200, got %d, request dump: %v", res.StatusCode, *res)
				}
				return nil
			})

			if err := g.Wait(); err != nil {
				t.Error(err)
			}
		}
	})

}

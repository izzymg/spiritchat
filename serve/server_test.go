package serve

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"spiritchat/config"
	"spiritchat/data"
	"testing"
	"time"

	"github.com/jackc/pgx/v4"
)

// Integrations - run on fake DB
func TestIntegration(t *testing.T) {
	conf, shouldRun := config.GetIntegrationsConfig()
	if !shouldRun {
		t.Log("Skipping integration test...")
		return
	}

	store, err := data.NewDatastore(context.Background(), conf.PGURL, conf.RedisURL, 1)
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

		client := &http.Client{
			Timeout: time.Second * 5,
		}
		for i := 0; i < 300; i++ {
			res, err := client.Post("http://"+conf.HTTPAddress+"/v1/"+catName+"/0", "application/JSON", bytes.NewReader(body))
			if err != nil {
				t.Errorf("writeThread got unexpected http error: %v", err)
			}
			io.Copy(ioutil.Discard, res.Body)
			res.Body.Close()
			if res.StatusCode != 200 {
				t.Errorf("writeThread expected status code 200, got %d, request dump: %v", res.StatusCode, *res)
			}
		}
	})

}

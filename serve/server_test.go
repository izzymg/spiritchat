package serve

import (
	"context"
	"net/http"
	"os"
	"spiritchat/data"
	"testing"
)

// Benchmark - integrations - run on fake DB

func BenchmarkGetCategories(b *testing.B) {

	b.StopTimer()

	pgURL := os.Getenv("SPIRITTEST_PG_URL")
	redisURL := os.Getenv("SPIRITTEST_REDIS_URL")
	addr := os.Getenv("SPIRITTEST_ADDR")
	if len(pgURL) == 0 || len(redisURL) == 0 || len(addr) == 0 {
		panic("SPIRITTEST_PG_URL or SPIRITTEST_REDIS_URL or SPIRITTEST_ADDR empty")
	}

	store, err := data.NewDatastore(
		context.Background(),
		pgURL, redisURL,
	)
	if err != nil {
		panic(err)
	}

	defer store.Cleanup(context.Background())

	server := NewServer(store, addr, "*")

	go func() {
		err := server.Listen(context.Background())
		if err != nil {
			panic(err)
		}
	}()
	b.StartTimer()

	_, err = http.Get("http://" + addr + "/v1")
	if err != nil {
		panic(err)
	}

	b.StopTimer()
}

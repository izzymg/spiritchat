package data

import (
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
)

// Integration test
func TestRateLimiting(t *testing.T) {
	t.Parallel()
	ip := "1.2.3.444"

	store := Store{
		redisPool: &redis.Pool{
			Dial: func() (redis.Conn, error) {
				conn, err := redis.DialURL("redis://localhost")
				if err != nil {
					t.Fatal(err)
				}
				return conn, nil
			},
		},
	}
	defer store.redisPool.Close()

	// Random key should not be limited
	limited, err := store.IsRateLimited("lllllllsssszzzz")
	if err != nil {
		t.Fatal(err)
	}
	if limited {
		t.Fatal("Expected limited == false")
	}

	store.RateLimit(ip, 2)
	limited, err = store.IsRateLimited(ip)
	if err != nil {
		t.Fatal(err)
	}
	if !limited {
		t.Fatal("Expected limited == true")
	}

	<-time.After(time.Second * 3)
	limited, err = store.IsRateLimited(ip)
	if err != nil {
		t.Fatal(err)
	}
	if limited {
		t.Fatal("Expected limited == false")
	}

}

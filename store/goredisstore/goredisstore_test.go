package goredisstore_test

import (
	"log"
	"testing"
	"time"

	"github.com/go-redis/redis"
	"github.com/throttled/throttled/v2"
	"github.com/throttled/throttled/v2/store/goredisstore"
	"github.com/throttled/throttled/v2/store/storetest"
)

const (
	redisTestDB     = 1
	redisTestPrefix = "throttled-go-redis:"
)

// Demonstrates that how to initialize a RateLimiter with redis
// using go-redis library.
func ExampleNew() {
	// import "github.com/go-redis/redis"

	// Initialize a redis client using go-redis
	client := redis.NewClient(&redis.Options{
		PoolSize:    10, // default
		IdleTimeout: 30 * time.Second,
		Addr:        "localhost:6379",
		Password:    "", // no password set
		DB:          0,  // use default DB
	})

	// Setup store
	store, err := goredisstore.NewCtx(client, "throttled:")
	if err != nil {
		log.Fatal(err)
	}

	// Setup quota
	quota := throttled.RateQuota{MaxRate: throttled.PerMin(20), MaxBurst: 5}

	// Then, use store and quota as arguments for NewGCRARateLimiter()
	throttled.NewGCRARateLimiterCtx(store, quota)
}

func TestRedisStore(t *testing.T) {
	c, st := setupRedis(t, 0)
	defer c.Close()
	defer clearRedis(c)

	clearRedis(c)
	storetest.TestGCRAStoreCtx(t, st)
	storetest.TestGCRAStoreTTLCtx(t, st)
}

func BenchmarkRedisStore(b *testing.B) {
	c, st := setupRedis(b, 0)
	defer c.Close()
	defer clearRedis(c)

	storetest.BenchmarkGCRAStoreCtx(b, st)
}

func clearRedis(c *redis.Client) error {
	keys, err := c.Keys(redisTestPrefix + "*").Result()
	if err != nil {
		return err
	}

	return c.Del(keys...).Err()
}

func setupRedis(tb testing.TB, ttl time.Duration) (*redis.Client, throttled.GCRAStoreCtx) {
	client := redis.NewClient(&redis.Options{
		PoolSize:    10, // default
		IdleTimeout: 30 * time.Second,
		Addr:        "localhost:6379",
		Password:    "",          // no password set
		DB:          redisTestDB, // use default DB
	})

	if err := client.Ping().Err(); err != nil {
		client.Close()
		tb.Skip("redis server not available on localhost port 6379")
	}

	st, err := goredisstore.NewCtx(client, redisTestPrefix)
	if err != nil {
		client.Close()
		tb.Fatal(err)
	}

	return client, st
}

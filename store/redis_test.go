package store

import (
	"testing"
	"time"

	"github.com/garyburd/redigo/redis"
)

const (
	redisTestDB     = 1
	redisTestPrefix = "throttled:"
)

func getPool() *redis.Pool {
	pool := &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 30 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", ":6379")
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
	return pool
}

func TestRedisStoreEval(t *testing.T) {
	c, st := setupRedis(t, true, 0)
	defer c.Close()
	defer clearRedis(c)

	clearRedis(c)
	storeTest(t, st)
	storeTTLTest(t, st)
}

func TestRedisStoreWatch(t *testing.T) {
	c, st := setupRedis(t, false, 0)
	defer c.Close()
	defer clearRedis(c)

	clearRedis(c)
	storeTest(t, st)
	storeTTLTest(t, st)
}

func BenchmarkRedisStoreEval(b *testing.B) {
	c, st := setupRedis(b, true, 0)
	defer c.Close()
	defer clearRedis(c)

	storeBenchmark(b, st)
}

func BenchmarkRedisStoreWatch(b *testing.B) {
	c, st := setupRedis(b, false, 0)
	defer c.Close()
	defer clearRedis(c)

	storeBenchmark(b, st)
}

func clearRedis(c redis.Conn) error {
	keys, err := redis.Values(c.Do("KEYS", redisTestPrefix+"*"))
	if err != nil {
		return err
	}

	if _, err := redis.Int(c.Do("DEL", keys...)); err != nil {
		return err
	}

	return nil
}

func setupRedis(tb testing.TB, useEval bool, ttl time.Duration) (redis.Conn, GCRAStore) {
	pool := getPool()
	c := pool.Get()

	if _, err := redis.String(c.Do("PING")); err != nil {
		c.Close()
		tb.Skip("redis server not available on localhost port 6379")
	}

	if _, err := redis.String(c.Do("SELECT", redisTestDB)); err != nil {
		c.Close()
		tb.Fatal(err)
	}

	st := NewRedisStore(pool, redisTestPrefix, redisTestDB)

	st.(*redisStore).supportsEval = useEval

	return c, st
}

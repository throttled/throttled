package radixstore_test

import (
	"testing"
	"time"

	"github.com/mediocregopher/radix/v3"
	"github.com/throttled/throttled/store/radixstore"
	"github.com/throttled/throttled/store/storetest"
)

const (
	redisTestDB     = 1
	redisTestPrefix = "throttled:"
)

func getPool(tb testing.TB) *radix.Pool {
	customConnFunc := func(network, addr string) (radix.Conn, error) {
		return radix.Dial(network, addr,
			radix.DialTimeout(30*time.Second),
			radix.DialSelectDB(redisTestDB),
		)
	}

	pool, err := radix.NewPool(
		"tcp",
		"127.0.0.1:6379",
		3,
		radix.PoolConnFunc(customConnFunc),
	)
	if err != nil {
		tb.Fatal(err)
	}

	return pool
}

func TestRedisStore(t *testing.T) {
	p, st := setupRedis(t, 0)
	defer p.Close()
	defer clearRedis(p)

	clearRedis(p)
	storetest.TestGCRAStore(t, st)
	storetest.TestGCRAStoreTTL(t, st)
}

func BenchmarkRedisStore(b *testing.B) {
	p, st := setupRedis(b, 0)
	defer p.Close()
	defer clearRedis(p)

	storetest.BenchmarkGCRAStore(b, st)
}

func clearRedis(p *radix.Pool) error {
	var keys []string
	if err := p.Do(radix.Cmd(&keys, "KEYS", redisTestPrefix+"*")); err != nil {
		return err
	}

	if err := p.Do(radix.Cmd(nil, "DEL", keys...)); err != nil {
		return err
	}

	return nil
}

func setupRedis(
	tb testing.TB, ttl time.Duration,
) (*radix.Pool, *radixstore.RadixStore) {
	pool := getPool(tb)

	if err := pool.Do(radix.Cmd(nil, "PING")); err != nil {
		pool.Close()
		tb.Skip("redis server not available on localhost port 6379")
	}

	st, err := radixstore.New(pool, redisTestPrefix)
	if err != nil {
		pool.Close()
		tb.Fatal(err)
	}

	return pool, st
}

package goredisstore_test

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/throttled/throttled/v2/store/goredisstore.v9"
)

func TestRedisStoreSetIfNotExistsWithTTL(t *testing.T) {
	for _, tt := range []struct {
		ttl  time.Duration
		want time.Duration
	}{
		{-time.Second, time.Second},
		{-time.Nanosecond, time.Second},
		{0, time.Second},
		{time.Nanosecond, time.Second},
		{999 * time.Millisecond, time.Second},
		{time.Second, time.Second},
		{1999 * time.Millisecond, time.Second},
		{2 * time.Second, 2 * time.Second},
		{2999 * time.Millisecond, 2 * time.Second},
		{time.Duration(1<<63 - 1), 9223372036 * time.Second},
	} {
		t.Run(tt.ttl.String(), func(t *testing.T) {
			c, st, key := setupRedisTTLTest(t)
			ctx := context.Background()

			if set, err := st.SetIfNotExistsWithTTL(ctx, "key", 42, tt.ttl); err != nil || !set {
				t.Fatalf("expected a new value to be set, got set=%v, err=%v", set, err)
			}
			if got, err := c.PTTL(ctx, key).Result(); err != nil || got > tt.want || got < tt.want-500*time.Millisecond {
				t.Fatalf("expected TTL near %v, got %v, err=%v", tt.want, got, err)
			}
			if got, err := c.Get(ctx, key).Int64(); err != nil || got != 42 {
				t.Fatalf("expected value 42, got %d, err=%v", got, err)
			}
		})
	}
}

func TestRedisStoreSetIfNotExistsWithTTLExistingKey(t *testing.T) {
	for _, initialTTL := range []time.Duration{time.Hour, 0} {
		t.Run(initialTTL.String(), func(t *testing.T) {
			c, st, key := setupRedisTTLTest(t)
			ctx := context.Background()

			if err := c.Set(ctx, key, 42, initialTTL).Err(); err != nil {
				t.Fatal(err)
			}
			if set, err := st.SetIfNotExistsWithTTL(ctx, "key", 99, 2*time.Second); err != nil || set {
				t.Fatalf("expected the existing key to be left unchanged, got set=%v, err=%v", set, err)
			}
			if got, err := c.Get(ctx, key).Int64(); err != nil || got != 42 {
				t.Fatalf("expected value 42, got %d, err=%v", got, err)
			}

			got, err := c.PTTL(ctx, key).Result()
			if err != nil {
				t.Fatal(err)
			}
			if initialTTL == 0 {
				if got != -1 {
					t.Fatalf("expected the key to remain without an expiry, got TTL %v", got)
				}
			} else if got < initialTTL-time.Second || got > initialTTL {
				t.Fatalf("expected TTL near %v to be preserved, got %v", initialTTL, got)
			}
		})
	}
}

func TestRedisStoreSetIfNotExistsWithTTLConcurrentCAS(t *testing.T) {
	c, st, key := setupRedisTTLTest(t)
	ctx := context.Background()
	other := redis.NewClient(c.Options())
	defer other.Close()
	otherStore, err := goredisstore.NewCtx(other, "")
	if err != nil {
		t.Fatal(err)
	}

	// Run a CAS on another client immediately after creation. With separate
	// SETNX and EXPIRE commands, this puts the CAS between them, so EXPIRE
	// overwrites the longer TTL. An atomic SET with expiry preserves it.
	called := false
	c.AddHook(afterCreateHook{key: key, after: func() {
		called = true
		if swapped, err := otherStore.CompareAndSwapWithTTL(ctx, key, 42, 99, time.Hour); err != nil || !swapped {
			t.Fatalf("expected the interleaved CAS to succeed, got swapped=%v, err=%v", swapped, err)
		}
	}})

	if set, err := st.SetIfNotExistsWithTTL(ctx, "key", 42, 2*time.Second); err != nil || !set {
		t.Fatalf("expected a new value to be set, got set=%v, err=%v", set, err)
	}
	if !called {
		t.Fatal("CAS was not interleaved with key creation")
	}
	if got, err := c.Get(ctx, key).Int64(); err != nil || got != 99 {
		t.Fatalf("expected CAS value 99, got %d, err=%v", got, err)
	}
	if got, err := c.PTTL(ctx, key).Result(); err != nil || got < time.Hour-time.Second || got > time.Hour {
		t.Fatalf("expected the one-hour CAS TTL to be preserved, got %v, err=%v", got, err)
	}
}

// setupRedisTTLTest uses a separate prefix so other adapters' tests cannot
// delete these keys when clearing their shared redisTestPrefix.
func setupRedisTTLTest(t *testing.T) (*redis.Client, *goredisstore.GoRedisStore, string) {
	t.Helper()
	c, _ := setupRedis(t, 0)
	t.Cleanup(func() { c.Close() })

	prefix := "throttled-go-redis-v9:" + t.Name() + ":"
	st, err := goredisstore.NewCtx(c, prefix)
	if err != nil {
		t.Fatal(err)
	}
	key := prefix + "key"
	if err := c.Del(context.Background(), key).Err(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Del(context.Background(), key) })
	return c, st, key
}

type afterCreateHook struct {
	key   string
	after func()
}

func (h afterCreateHook) DialHook(next redis.DialHook) redis.DialHook { return next }

func (h afterCreateHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

func (h afterCreateHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		err := next(ctx, cmd)
		if err == nil && (cmd.Name() == "set" || cmd.Name() == "setnx") && cmd.Args()[1] == h.key {
			h.after()
		}
		return err
	}
}

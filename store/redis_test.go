package store

import (
	"testing"
	"time"

	"github.com/PuerkitoBio/throttled"
	"github.com/garyburd/redigo/redis"
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

func TestRedisStore(t *testing.T) {
	pool := getPool()
	st := NewRedisStore(pool, "throttled:", 1)

	// Get invalid key returns NoSuchKey
	_, _, err := st.(throttled.StoreSecs).GetSecs("unknown")
	if err != throttled.ErrNoSuchKey {
		t.Errorf("expected get of unknown key to return %s, got %v", throttled.ErrNoSuchKey, err)
	}

	// Reset stores a key with count of 1, full window remaining
	window := 5 * time.Second
	err = st.Reset("k", window)
	if err != nil {
		t.Errorf("expected reset to return nil, got %s", err)
	}
	cnt, sec1, _ := st.(throttled.StoreSecs).GetSecs("k")
	if cnt != 1 {
		t.Errorf("expected reset to set count to 1, got %d", cnt)
	}
	if sec1 != int(window.Seconds()) {
		t.Errorf("expected remaining seconds to be %d, got %d", int(window.Seconds()), sec1)
	}

	// Incr increments the key
	cnt, err = st.Incr("k")
	if err != nil {
		t.Errorf("expected incr to return nil error, got %s", err)
	}
	if cnt != 2 {
		t.Errorf("expected incr to return 2, got %d", cnt)
	}
	cnt, sec2, _ := st.(throttled.StoreSecs).GetSecs("k")
	if cnt != 2 {
		t.Errorf("expected cnt after incr to return 2, got %d", cnt)
	}
	if sec2 != sec1 {
		t.Errorf("expected to get same remaining seconds %d, got %d", sec1, sec2)
	}

	// Waiting a second diminishes the remaining seconds
	time.Sleep(time.Second)
	_, sec3, _ := st.(throttled.StoreSecs).GetSecs("k")
	if sec3 != sec1-1 {
		t.Errorf("expected get after a 1s sleep to return %d remaining seconds, got %d", sec1-1, sec3)
	}

	// Reset on existing key brings it back to 1, new timestamp
	err = st.Reset("k", time.Second)
	if err != nil {
		t.Errorf("expected reset on existing key to return nil, got %s", err)
	}
	cnt, sec4, _ := st.(throttled.StoreSecs).GetSecs("k")
	if cnt != 1 {
		t.Errorf("expected reset on existing key to return cnt of 1, got %d", cnt)
	}
	if sec4 != 1 {
		t.Errorf("expected reset to return remaining seconds of %d, got %d", 1, sec4)
	}

	// Waiting a second so the key expires, Get should return no such key
	time.Sleep(1100 * time.Millisecond)
	_, _, err = st.(throttled.StoreSecs).GetSecs("k")
	if err != throttled.ErrNoSuchKey {
		t.Errorf("expected get after key expiration to return %s, got %v", throttled.ErrNoSuchKey, err)
	}
}

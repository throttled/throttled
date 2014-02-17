package store

import (
	"testing"
	"time"

	"github.com/PuerkitoBio/throttled"
)

func TestMemStore(t *testing.T) {
	st := NewMemStore()

	// Get invalid key returns NoSuchKey
	_, _, err := st.(throttled.StoreTs).GetTs("unknown")
	if err != throttled.ErrNoSuchKey {
		t.Errorf("expected get of unknown key to return %s, got %v", throttled.ErrNoSuchKey, err)
	}

	// Reset stores a key with count of 1, current timestamp
	err = st.Reset("k", time.Second)
	if err != nil {
		t.Errorf("expected reset to return nil, got %s", err)
	}
	cnt, ts1, _ := st.(throttled.StoreTs).GetTs("k")
	if cnt != 1 {
		t.Errorf("expected reset to set count to 1, got %d", cnt)
	}

	// Incr increments the key, keeps same timestamp
	cnt, err = st.Incr("k")
	if err != nil {
		t.Errorf("expected incr to return nil error, got %s", err)
	}
	if cnt != 2 {
		t.Errorf("expected incr to return 2, got %d", cnt)
	}
	cnt, ts2, _ := st.(throttled.StoreTs).GetTs("k")
	if cnt != 2 {
		t.Errorf("expected cnt after incr to return 2, got %d", cnt)
	}
	if !ts2.Equal(ts1) {
		t.Errorf("expected get to return initial timestamp %s, got %s", ts1, ts2)
	}

	// Reset on existing key brings it back to 1, new timestamp
	err = st.Reset("k", time.Second)
	if err != nil {
		t.Errorf("expected reset on existing key to return nil, got %s", err)
	}
	cnt, ts3, _ := st.(throttled.StoreTs).GetTs("k")
	if cnt != 1 {
		t.Errorf("expected reset on existing key to return cnt of 1, got %d", cnt)
	}
	if !ts3.After(ts1) {
		t.Errorf("expected reset to set new timestamp after %s, got %s", ts1, ts3)
	}
}

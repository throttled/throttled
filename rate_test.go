package throttled

import (
	"testing"
	"time"

	"gopkg.in/throttled/throttled.v0/store"
)

const deniedStatus = 429

type testStore struct {
	store store.GCRAStore

	clock       time.Time
	failUpdates bool
}

func (ts *testStore) GetWithTime(key string) (int64, time.Time, error) {
	v, _, e := ts.store.GetWithTime(key)
	return v, ts.clock, e
}

func (ts *testStore) SetIfNotExists(key string, value int64, ttl time.Duration) (bool, error) {
	if ts.failUpdates {
		return false, nil
	}
	return ts.store.SetIfNotExists(key, value, ttl)
}

func (ts *testStore) CompareAndSwap(key string, old, new int64, ttl time.Duration) (bool, error) {
	if ts.failUpdates {
		return false, nil
	}
	return ts.store.CompareAndSwap(key, old, new, ttl)
}

func TestRateLimit(t *testing.T) {
	limit := 5
	rq := RateQuota{limit, 5 * time.Second}
	start := time.Unix(0, 0)
	cases := []struct {
		now          time.Time
		remaining    int
		reset, retry time.Duration
		limited      bool
	}{
		0:  {start, 4, time.Second, -1, false},
		1:  {start, 3, 2 * time.Second, -1, false},
		2:  {start, 2, 3 * time.Second, -1, false},
		3:  {start, 1, 4 * time.Second, -1, false},
		4:  {start, 0, 5 * time.Second, -1, false},
		5:  {start, 0, 5 * time.Second, time.Second, true},
		6:  {start.Add(3000 * time.Millisecond), 2, 3000 * time.Millisecond, -1, false},
		7:  {start.Add(3100 * time.Millisecond), 1, 3900 * time.Millisecond, -1, false},
		8:  {start.Add(4000 * time.Millisecond), 1, 4000 * time.Millisecond, -1, false},
		9:  {start.Add(8000 * time.Millisecond), 4, 1000 * time.Millisecond, -1, false},
		10: {start.Add(9500 * time.Millisecond), 4, 1000 * time.Millisecond, -1, false},
	}

	st := testStore{store: store.NewMemStore(0)}
	rl, err := NewGCRARateLimiter(&st, rq)
	if err != nil {
		t.Fatal(err)
	}

	// Start the server
	for i, c := range cases {
		st.clock = c.now

		lr, err := rl.Limit("foo")
		if err != nil {
			t.Fatalf("%d: %#v", i, err)
		}

		if have, want := lr.Limited(), c.limited; have != want {
			t.Errorf("%d: expected Limited to be %t but got %t", i, want, have)
		}

		rlr := lr.(RateLimitResult)

		if have, want := rlr.Limit(), limit; have != want {
			t.Errorf("%d: expected Limit to be %d but got %d", i, want, have)
		}

		if have, want := rlr.Remaining(), c.remaining; have != want {
			t.Errorf("%d: expected Remaining to be %d but got %d", i, want, have)
		}

		if have, want := rlr.Reset(), c.reset; have != want {
			t.Errorf("%d: expected Reset to be %s but got %s", i, want, have)
		}

		if have, want := rlr.RetryAfter(), c.retry; have != want {
			t.Errorf("%d: expected RetryAfter to be %d but got %d", i, want, have)
		}
	}
}

func TestRateLimitUpdateFailures(t *testing.T) {
	rq := RateQuota{1, time.Second}
	st := testStore{store: store.NewMemStore(0), failUpdates: true}
	rl, err := NewGCRARateLimiter(&st, rq)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := rl.Limit("foo"); err == nil {
		t.Error("Expected limiting to fail when store updates fail")
	}
}

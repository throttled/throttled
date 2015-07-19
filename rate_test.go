package throttled

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/PuerkitoBio/throttled/store"
)

const deniedStatus = 429

func TestRateLimit(t *testing.T) {
	limit := 5
	quota := Q{limit, 5 * time.Second}
	cases := []struct {
		now, remain, reset, retry, status int
	}{
		0: {0, 4, 1, 0, 200},
		1: {0, 3, 2, 0, 200},
		2: {0, 2, 3, 0, 200},
		3: {0, 1, 4, 0, 200},
		4: {0, 0, 5, 0, 200},
		5: {0, 0, 5, 1, deniedStatus},
		6: {3000, 2, 3, 0, 200},
		7: {3100, 1, 4, 0, 200},
		8: {4000, 1, 4, 0, 200},
		9: {8000, 4, 1, 0, 200},
	}

	ms, err := store.NewMemStore(0)
	if err != nil {
		t.Fatal(err)
	}

	// Rate limit
	rl := RateLimit(quota, nil, ms)
	// Create the stats
	st := &stats{}
	// Create the handler
	h := rl.Throttle(st)

	now := 0
	rl.limiter.(*rateLimiter).clock = func() int64 { return int64(now) }

	// Start the server
	srv := httptest.NewServer(h)
	defer srv.Close()
	for i, c := range cases {
		now = int(time.Millisecond) * c.now
		callRateLimited(t, i, limit, c.remain, c.reset, c.retry, c.status, srv.URL)
	}
}

func callRateLimited(t *testing.T, i, limit, remain, reset, retry, status int, url string) {
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	// Assert status code
	if status != res.StatusCode {
		t.Errorf("%d: expected status %d, got %d", i, status, res.StatusCode)
	}
	// Assert headers
	if v := res.Header.Get("X-RateLimit-Limit"); v != strconv.Itoa(limit) {
		t.Errorf("%d: expected limit header to be %d, got %s", i, limit, v)
	}
	if v := res.Header.Get("X-RateLimit-Remaining"); v != strconv.Itoa(remain) {
		t.Errorf("%d: expected remain header to be %d, got %s", i, remain, v)
	}
	// Allow 1 second wiggle room
	v := res.Header.Get("X-RateLimit-Reset")
	vi, _ := strconv.Atoi(v)
	if vi != reset {
		t.Errorf("%d: expected reset header to be %d, got %d", i, reset, vi)
	}
	if retry != 0 {
		v := res.Header.Get("Retry-After")
		vi, _ := strconv.Atoi(v)
		if vi != retry {
			t.Errorf("%d: expected reset header to be %d, got %d", i, reset, vi)
		}
	}
}

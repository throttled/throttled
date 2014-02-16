package throttled

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestRateLimit(t *testing.T) {
	quota := PerSec(5)
	//_, win := quota.Quota()
	cases := []struct {
		limit, remain, reset, status int
	}{
		0: {5, 4, 5, 200},
		1: {5, 3, 5, 200},
		2: {5, 2, 4, 200},
		3: {5, 1, 4, 200},
		4: {5, 0, 3, 200},
		5: {5, 0, 3, 503},
	}
	// Limit the requests to 2 per second
	th := Interval(PerSec(2), 10, nil)
	// Rate limit
	rl := RateLimit(quota, nil, NewMemStore())
	// Create the stats
	st := &stats{}
	// Create the handler
	h := th.Throttle(rl.Throttle(st))

	// Start the server
	srv := httptest.NewServer(h)
	defer srv.Close()
	for i, c := range cases {
		res, err := http.Get(srv.URL)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		// Assert status code
		if c.status != res.StatusCode {
			t.Errorf("%d: expected status %d, got %d", i, c.status, res.StatusCode)
		}
		// Assert headers
		if v := res.Header.Get("X-RateLimit-Limit"); v != strconv.Itoa(c.limit) {
			t.Errorf("%d: expected limit header to be %d, got %s", i, c.limit, v)
		}
		if v := res.Header.Get("X-RateLimit-Remaining"); v != strconv.Itoa(c.remain) {
			t.Errorf("%d: expected remain header to be %d, got %s", i, c.remain, v)
		}
	}
}

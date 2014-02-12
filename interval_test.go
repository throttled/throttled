package throttled

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

type stats struct {
	sync.Mutex
	ok      int
	dropped int
	ts      []time.Time
}

func (s *stats) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Lock()
	defer s.Unlock()
	s.ts = append(s.ts, time.Now())
	s.ok++
	w.WriteHeader(200)
}

func (s *stats) DroppedHTTP(w http.ResponseWriter, r *http.Request) {
	s.Lock()
	defer s.Unlock()
	s.dropped++
	w.WriteHeader(503)
}

func TestInterval(t *testing.T) {
	cases := []struct {
		rate     Delayer
		bursts   int
		handlers int
		min      int
		max      int
	}{
		0: {PerSec(50), 3, 6, 3, 5},
		1: {PerSec(20), 0, 1, 1, 1},
		2: {PerSec(30), 0, 3, 1, 2},
		3: {PerSec(4), 2, 6, 2, 4},
		4: {PerSec(0), 0, 6, 1, 6},
		5: {PerSec(0), 10, 6, 6, 6},
	}
	for i, c := range cases {
		func() {
			var (
				resps []int
				mu    sync.Mutex
			)
			// Configure the throttler
			th := Interval(c.rate, c.bursts)
			st := &stats{}
			th.DroppedHandler = http.HandlerFunc(st.DroppedHTTP)
			hn := th.Throttle(st)
			// Start the web server
			srv := httptest.NewServer(hn)
			defer srv.Close()
			// Launch the requests
			for j := 0; j < c.handlers; j++ {
				go func() {
					res, err := http.Get(srv.URL)
					if err != nil {
						panic(err)
					}
					mu.Lock()
					defer mu.Unlock()
					resps = append(resps, res.StatusCode)
				}()
			}
			// Wait for the requests to complete
			time.Sleep(100*time.Millisecond + (c.rate.Delay() * time.Duration(c.handlers)))
			st.Lock()
			defer st.Unlock()
			// Test that the number of OK calls are within min and max
			if st.ok < c.min || st.ok > c.max {
				t.Errorf("%d: expected between %d and %d calls, got %d", i, c.min, c.max, st.ok)
			}
			// The number of dropped calls should balance
			if expdrop := (c.handlers - st.ok); st.dropped != expdrop {
				t.Errorf("%d: expected %d dropped, got %d", i, expdrop, st.dropped)
			}
			// Test that the timestamps are separated by the rate's delay
			for j := 0; j < len(st.ts)-1; j++ {
				if (st.ts[j+1].Sub(st.ts[j]) < c.rate.Delay()) || (st.ts[j+1].Sub(st.ts[j]) > c.rate.Delay()+50*time.Millisecond) {
					t.Errorf("%d: expected calls to be %s apart", i, c.rate.Delay())
				}
			}
			// Test that the right status codes have been received
			twos, fives := 0, 0
			mu.Lock()
			defer mu.Unlock()
			for j := 0; j < len(resps); j++ {
				if resps[j] == 200 {
					twos++
				} else if resps[j] == 503 {
					fives++
				} else {
					t.Errorf("%d: unexpected status code: %d", i, resps[j])
				}
			}
			if twos != st.ok {
				t.Errorf("%d: expected %d status 200, got %d", i, st.ok, twos)
			}
			if fives != (c.handlers - st.ok) {
				t.Errorf("%d: expected %d status 503, got %d", i, c.handlers-st.ok, fives)
			}
		}()
	}
}

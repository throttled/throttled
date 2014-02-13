package throttled

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

type stats struct {
	sync.Mutex
	ok      int
	dropped int
	ts      []time.Time

	body func()
}

func (s *stats) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.body != nil {
		s.body()
	}
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

func (s *stats) Stats() (int, int, []time.Time) {
	s.Lock()
	defer s.Unlock()
	return s.ok, s.dropped, s.ts
}

func runTest(th *Throttler, calls int, wait time.Duration, body func()) (*stats, []*http.Response) {
	st := &stats{body: body}
	th.DroppedHandler = http.HandlerFunc(st.DroppedHTTP)
	hn := th.Throttle(st)
	resps := runTestHandler(hn, calls, wait)
	th.Close()
	return st, resps
}

func runTestHandler(h http.Handler, calls int, wait time.Duration) []*http.Response {
	var mu sync.Mutex
	var resps []*http.Response

	// Start the web server
	srv := httptest.NewServer(h)
	defer srv.Close()
	// Launch the requests
	for j := 0; j < calls; j++ {
		go func() {
			res, err := http.Get(srv.URL)
			if err != nil {
				panic(err)
			}
			defer res.Body.Close()
			mu.Lock()
			defer mu.Unlock()
			resps = append(resps, res)
		}()
	}
	time.Sleep(wait)
	mu.Lock()
	defer mu.Unlock()
	return resps
}

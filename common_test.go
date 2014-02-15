package throttled

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"github.com/PuerkitoBio/boom/commands"
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

func runTest(h http.Handler, b commands.Boom) *commands.Report {
	srv := httptest.NewServer(h)
	defer srv.Close()
	b.Req.Url = srv.URL
	return b.Run()
}

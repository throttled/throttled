package throttled

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestThrottleFuncCall(t *testing.T) {
	ok := false
	th := New(PerSec(1), 0)
	fn := th.Func(func() {
		ok = true
	})
	fn()

	if !ok {
		t.Error("expected func to be called")
	}
}

type stats struct {
	sync.Mutex
	fnok      int
	fndropped int
	webres    []*http.Response
	ts        []time.Time
}

func (s *stats) Fnok() {
	s.Lock()
	defer s.Unlock()
	s.fnok++
	s.ts = append(s.ts, time.Now())
}

func (s *stats) Fndropped() {
	s.Lock()
	defer s.Unlock()
	s.fndropped++
}

func (s *stats) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Lock()
	defer s.Unlock()
	s.ts = append(s.ts, time.Now())
	s.fnok++
	w.WriteHeader(200)
}

func (s *stats) DroppedHTTP(w http.ResponseWriter, r *http.Request) {
	s.Lock()
	defer s.Unlock()
	s.fndropped++
	w.WriteHeader(503)
}

func TestThrottle(t *testing.T) {
	cases := []struct {
		rate     Delayer
		bursts   int
		funcs    int
		handlers int
		min      int
		max      int
	}{
		0: {PerSec(4), 3, 6, 0, 3, 5},
		1: {PerSec(4), 0, 1, 0, 1, 1},
		2: {PerSec(10), 1, 5, 0, 1, 3},
		3: {PerSec(4), 3, 0, 6, 3, 5},
		4: {PerSec(10), 5, 4, 4, 5, 7},
	}
	var (
		mu   sync.Mutex
		ares []*http.Response
	)
	for i, c := range cases {
		th := New(c.rate, c.bursts)
		st := &stats{}
		if c.funcs > 0 {
			fn := th.FuncDropped(st.Fnok, st.Fndropped)
			for j := 0; j < c.funcs; j++ {
				go fn()
			}
		}
		if c.handlers > 0 {
			hn := th.HandlerDropped(st, http.HandlerFunc(st.DroppedHTTP))
			srv := httptest.NewServer(hn)
			for j := 0; j < c.handlers; j++ {
				go func() {
					res, err := http.Get(srv.URL)
					if err != nil {
						panic(err)
					}
					mu.Lock()
					defer mu.Unlock()
					ares = append(ares, res)
				}()
			}
		}
		time.Sleep(time.Second + (c.rate.Delay() * time.Duration(c.funcs+c.handlers)))
		st.Lock()
		defer st.Unlock()
		// Test that the number of OK calls are within min and max
		if st.fnok < c.min || st.fnok > c.max {
			t.Errorf("%d: expected between %d and %d calls, got %d", i, c.min, c.max, st.fnok)
		}
		if expdrop := (c.funcs + c.handlers - st.fnok); st.fndropped != expdrop {
			t.Errorf("%d: expected %d dropped, got %d", i, expdrop, st.fndropped)
		}
		// Test that the timestamps are separated by the rate's delay
		for j := 0; j < len(st.ts)-1; j++ {
			if (st.ts[j+1].Sub(st.ts[j]) < c.rate.Delay()) || (st.ts[j+1].Sub(st.ts[j]) > c.rate.Delay()+50*time.Millisecond) {
				t.Errorf("%d: expected calls to be %s apart", i, c.rate.Delay())
			}
		}
	}
}

package throttled

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"
)

// X-RateLimit-Limit : Quota
// X-RateLimit-Remaining : number of requests remaining in the current window
// X-RateLimit-Reset : seconds before a new window

func RateLimit(q Quota, vary *VaryBy, store Store) *Throttler {
	var l Limiter
	reqs, win := q.Quota()
	if vary == nil {
		l = &rateLimiter{
			reqs:   reqs,
			window: win,
			store:  store,
		}
	}
	return &Throttler{
		limiter: l,
	}
}

// TODO : Use a rateLimiter for vary = nil, and rateLimiterVary otherwise?
type rateLimiter struct {
	reqs   int
	window time.Duration
	store  Store
}

func (r *rateLimiter) Start() {
	if r.reqs < 1 {
		r.reqs = 1
	}
}

func (r *rateLimiter) Request(w http.ResponseWriter, req *http.Request) (<-chan bool, error) {
	var cnt int
	var secs float64
	var err error
	var ts time.Time

	ch := make(chan bool, 1)
	// Get the current count and remaining seconds
	switch st := r.store.(type) {
	case StoreTs:
		cnt, ts, err = st.GetTs("")
		fmt.Println("cnt: ", cnt, "ts: ", ts, "diff: ", r.window-time.Now().UTC().Sub(ts))
		if !ts.IsZero() {
			secs = (r.window - time.Now().UTC().Sub(ts)).Seconds()
		}
	case StoreSecs:
		cnt, secs, err = st.GetSecs("")
	}
	fmt.Println("cnt: ", cnt, "secs: ", secs)
	// If error getting the current count, return
	if err != nil {
		return nil, err
	}
	if secs > 0 && cnt > r.reqs {
		// Still in limited window, and too many requests, deny
		ch <- false
	} else if secs <= 0 {
		// New limited window starting, reset the count, allow
		if err := r.store.Reset("", r.window); err != nil {
			return nil, err
		}
		cnt = 1
		secs = r.window.Seconds()
		ch <- true
	} else {
		// Still in limited window, requests remaining, increment and allow
		cnt, err = r.store.Incr("")
		if err != nil {
			return nil, err
		}
		ch <- true
	}
	// Set rate-limit headers
	w.Header().Add("X-RateLimit-Limit", strconv.Itoa(r.reqs))
	w.Header().Add("X-RateLimit-Remaining", strconv.Itoa(int(math.Max(float64(r.reqs-cnt), 0))))
	w.Header().Add("X-RateLimit-Reset", strconv.FormatFloat(secs, 'f', 0, 64))
	return ch, nil
}

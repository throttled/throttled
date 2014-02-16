package throttled

import (
	"math"
	"net/http"
	"strconv"
	"time"
)

// X-RateLimit-Limit : Quota
// X-RateLimit-Remaining : number of requests remaining in the current window
// X-RateLimit-Reset : seconds before a new window

func RateLimit(q Quota, vary *VaryBy, store Store) *Throttler {
	return nil
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
	if r.store == nil {
		r.store = DefaultStore
	}
	r.store.Init(r.reqs, r.window)
}

func (r *rateLimiter) Request(w http.ResponseWriter, req *http.Request) (<-chan bool, error) {
	ch := make(chan bool, 1)
	cnt, secs, err := r.store.Get("")
	if err != nil {
		return nil, err
	}
	if cnt > r.reqs {
		ch <- false
	} else {
		if err := r.store.Incr(""); err != nil {
			return nil, err
		}
		ch <- true
	}
	// Set rate-limit headers
	w.Header().Add("X-RateLimit-Limit", strconv.Itoa(r.reqs))
	w.Header().Add("X-RateLimit-Remaining", strconv.Itoa(int(math.Max(float64(r.reqs-cnt), 0))))
	if secs > 0 {
		w.Header().Add("X-RateLimit-Reset", strconv.Itoa(secs))
	} else {
		w.Header().Add("X-RateLimit-Reset", strconv.Itoa(int(r.window.Seconds())))
	}
	return ch, nil
}

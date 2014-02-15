package throttled

import (
	"net/http"
	"time"
)

// X-RateLimit-Limit : Quota
// X-RateLimit-Remaining : number of requests remaining in the current window
// X-RateLimit-Reset : seconds before a new window

func RateLimit(q Quota, vary *VaryBy, store Store) *Throttler {
	return nil
}

// TODO : Use a rateLimiter for vary = nil, and rateLimiterVary otherwise?
// TODO : Set headers before sending to drop handler, or use its own "drop" handler?
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
}

func (r *rateLimiter) Request(w http.ResponseWriter, req *http.Request) (<-chan bool, error) {
	ch := make(chan bool, 1)
	cnt, err := r.store.Get("")
	if err != nil {
		OnError(w, req, err)
	}
	if cnt > r.reqs {
	}
}

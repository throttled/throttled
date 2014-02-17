package throttled

import (
	"math"
	"net/http"
	"strconv"
	"time"
)

// RateLimit creates a throttler that limits the number of requests allowed
// in a certain time window defined by the Quota q. The q parameter specifies
// the requests per time window, and it is silently set to at least 1 request
// and at least a 1 second window if it is less than that. The time window
// starts when the first request is made outside an existing window.
//
// The vary parameter indicates what criteria should be used to group requests
// for which the limit must be applied (ex.: rate limit based on the remote address).
// See varyby.go for the various options.
//
// The specified store is used to keep track of the request count and the
// time remaining in the window. The throttled package comes with some stores
// in the throttled/store package. Custom stores can be created too, by implementing
// the Store interface.
//
// The rate limit throttler sets the following headers on the response:
//
//    X-RateLimit-Limit : quota
//    X-RateLimit-Remaining : number of requests remaining in the current window
//    X-RateLimit-Reset : seconds before a new window
//
// Additionally, if the request was denied access, the following header is added:
//
//    Retry-After : seconds before the caller should retry
//
func RateLimit(q Quota, vary *VaryBy, store Store) *Throttler {
	// Make sure the store implements a getter
	switch store.(type) {
	case StoreTs, StoreSecs:
	default:
		panic("throttled: store must implement either throttled.StoreTs or throttled.StoreSecs")
	}

	// Extract requests and window
	reqs, win := q.Quota()

	// Create and return the throttler
	return &Throttler{
		limiter: &rateLimiter{
			reqs:   reqs,
			window: win,
			vary:   vary,
			store:  store,
		},
	}
}

// The rate limiter implements limiting the request to a certain quota
// based on the vary-by criteria. State is saved in the store.
type rateLimiter struct {
	reqs   int
	window time.Duration
	vary   *VaryBy
	store  Store
}

// Start initializes the limiter for execution.
func (r *rateLimiter) Start() {
	if r.reqs < 1 {
		r.reqs = 1
	}
	if r.window < time.Second {
		r.window = time.Second
	}
}

// Request is called for each request to the throttled handler. It checks if
// the request can go through and signals it via the returned channel.
// It returns an error if the operation fails.
func (r *rateLimiter) Request(w http.ResponseWriter, req *http.Request) (<-chan bool, error) {
	var cnt, secs int
	var err error
	var ts time.Time

	// Create return channel and initialize
	ch := make(chan bool, 1)
	ok := true
	key := r.vary.Key(req)

	// Get the current count and remaining seconds
	switch st := r.store.(type) {
	case StoreTs:
		cnt, ts, err = st.GetTs(key)
		if err == nil {
			secs = int((r.window - time.Now().UTC().Sub(ts)).Seconds())
		}
	case StoreSecs:
		cnt, secs, err = st.GetSecs(key)
	}

	// Handle the possible situations: error, begin new window, or increment current window.
	switch {
	case err != nil && err != ErrNoSuchKey:
		// An unexpected error occurred
		return nil, err
	case err == ErrNoSuchKey || secs <= 0:
		// Reset counter
		if err := r.store.Reset(key, r.window); err != nil {
			return nil, err
		}
		cnt = 1
		secs = int(r.window.Seconds())
	default:
		// Increment
		cnt, err = r.store.Incr(key)
		if err != nil {
			return nil, err
		}
		// If the limit is reached, deny access
		if cnt > r.reqs {
			ok = false
		}
	}
	// Set rate-limit headers
	w.Header().Add("X-RateLimit-Limit", strconv.Itoa(r.reqs))
	w.Header().Add("X-RateLimit-Remaining", strconv.Itoa(int(math.Max(float64(r.reqs-cnt), 0))))
	w.Header().Add("X-RateLimit-Reset", strconv.Itoa(secs))
	if !ok {
		w.Header().Add("Retry-After", strconv.Itoa(secs))
	}
	// Send response via the return channel
	ch <- ok
	return ch, nil
}

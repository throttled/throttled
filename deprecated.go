package throttled

import (
	"net/http"
	"time"

	"gopkg.in/throttled/throttled.v0/store"
)

// DEPRECATED. Quota returns the number of requests allowed and the custom time window.
func (q RateQuota) Quota() (int, time.Duration) {
	return q.Count, q.Period
}

// DEPRECATED. Q represents a custom quota.
type Q struct {
	Requests int
	Window   time.Duration
}

// DEPRECATED. Quota returns the number of requests allowed and the custom time window.
func (q Q) Quota() (int, time.Duration) {
	return q.Requests, q.Window
}

// DEPRECATED. The Quota interface defines the method to implement to describe
// a time-window quota, as required by the RateLimit throttler.
type Quota interface {
	// Quota returns a number of requests allowed, and a duration.
	Quota() (int, time.Duration)
}

// DEPRECATED. Throttler is a backwards-compatible alias for HTTPLimiter.
type Throttler struct {
	HTTPRateLimiter
}

// DEPRECATED. Throttle is an alias for HTTPLimiter#Limit
func (t *Throttler) Throttle(h http.Handler) http.Handler {
	return t.RateLimit(h)
}

// DEPRECATED. RateLimit creates a Throttler that conforms to the given
// rate limits
func RateLimit(q Quota, vary *VaryBy, store store.GCRAStore) *Throttler {
	count, period := q.Quota()
	limiter, err := NewGCRARateLimiter(store, RateQuota{count, period})
	// TODO: It's sad to introduce this panic but I think better than disallowing
	// errors from the initialization.
	if err != nil {
		panic(err)
	}

	return &Throttler{
		HTTPRateLimiter{
			RateLimiter: limiter,
			VaryBy:      vary,
		},
	}
}

// DEPRECATED. Store is an alias for store.GCRAStore
type Store interface {
	store.GCRAStore
}

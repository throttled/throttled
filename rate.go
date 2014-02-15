package throttled

import "time"

// X-RateLimit-Limit : Quota
// X-RateLimit-Remaining : number of requests remaining in the current window
// X-RateLimit-Reset : seconds before a new window

func RateLimit(q Quota, vary *VaryBy) *Throttler {
	return RateLimitStore(q, vary, nil)
}

func RateLimitStore(q Quota, vary *VaryBy, store Store) *Throttler {
	return nil
}

// TODO : Use a rateLimiter for vary = nil, and rateLimiterVary otherwise?
// TODO : Set headers before sending to drop handler, or use its own "drop" handler?
type rateLimiter struct {
	reqs   int
	window time.Duration
	store  Store
}

package throttled

import (
	"time"

	"gopkg.in/throttled/throttled.v0/store"
)

// A RateLimitResult is returned by a Limiter to provide additional context
// about the state of rate limiting at the time of the query. This state can
// be used, for example, to communicate information to the client via HTTP headers.
// Any function may return a negative value to indicate that the particular
// attribute is not relevant to the implementation or state.
type RateLimitResult interface {
	LimitResult

	// Limit returns the maximum number of requests that could be permitted
	// instantaneously per key starting from an empty state. For example,
	// if a rate limiter allows 10 requests per second, Limit would always
	// return 10.
	Limit() int

	// Remaining returns the maximum number of requests that could be permitted
	// instantaneously per key given the current state. For example, if a rate
	// limiter allows 10 requests per second and has already received 6 requests
	// for a given key this second, Remaining would return 4.
	Remaining() int

	// Reset returns the time until the rate limiter returns to its initial
	// state for a given key. For example, if a rate limiter manages requests per second
	// per second and received one request 200ms ago, Reset would return 800ms.
	// This should be the earliest time when Limit and Remaining are equal.
	Reset() time.Duration

	// RetryAfter returns the time until the next request will be permitted.
	// It should only be set if the current request was limited.
	RetryAfter() time.Duration
}

type rateLimitResult struct {
	limited           bool
	limit, remaining  int
	reset, retryAfter time.Duration
}

func (r *rateLimitResult) Limited() bool             { return r.limited }
func (r *rateLimitResult) Limit() int                { return r.limit }
func (r *rateLimitResult) Remaining() int            { return r.remaining }
func (r *rateLimitResult) Reset() time.Duration      { return r.reset }
func (r *rateLimitResult) RetryAfter() time.Duration { return r.retryAfter }

// RateQuota describes the number of requests allowed per time period.
// The Count also represents the maximum number of requests permitted in
// a burst. For example, a quota of 60 requests every minute would allow either
// a continuous stream of 1 request every second or a burst of 60 requests at the
// same time once a minute.
type RateQuota struct {
	Count  int
	Period time.Duration
}

// PerSec represents a number of requests per second.
func PerSec(n int) RateQuota { return RateQuota{n, time.Second} }

// PerMin represents a number of requests per minute.
func PerMin(n int) RateQuota { return RateQuota{n, time.Minute} }

// PerHour represents a number of requests per hour.
func PerHour(n int) RateQuota { return RateQuota{n, time.Hour} }

// PerDay represents a number of requests per day.
func PerDay(n int) RateQuota { return RateQuota{n, 24 * time.Hour} }

type gcraLimiter struct {
}

// NewGCRARateLimiter creates a Limiter that uses the generic cell-rate
// algorithm. Calls to `Limit` return a `RateLimitResult`.
//
// quota.Count defines the maximum number of requests permitted in an
// instantaneous burst and quota.Count / quota.Period defines the maximum
// sustained rate. For example, PerMin(60) permits 60 requests instantly per key
// followed by one request per second indefinitely whereas PerSec(1) only permits
// one request per second with no tolerance for bursts.
func NewGCRARateLimiter(st store.GCRAStore, quota RateQuota) (Limiter, error) {
	return &gcraLimiter{}, nil
}

func (g *gcraLimiter) Limit(key string) (LimitResult, error) {
	return &rateLimitResult{}, nil
}

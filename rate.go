package throttled

import (
	"fmt"
	"time"

	"gopkg.in/throttled/throttled.v0/store"
)

const (
	// Maximum number of times to retry SetIfNotExists/CompareAndSwap operations
	// before returning an error.
	maxCASAttempts = 10
)

// The RateLimitResult interface is implemented by LimitResults to provide
// additional context about the state of rate limiting at the time of the query.
// This state can be used, for example, to communicate information to the client
// via HTTP headers. Any function may return a negative value to indicate that
// the particular attribute is not relevant to the implementation or state.
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

type limitResult struct {
	limited bool
}

func (r *limitResult) Limited() bool { return r.limited }

type rateLimitResult struct {
	limitResult

	limit, remaining  int
	reset, retryAfter time.Duration
}

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
	limit int
	// Think of the DVT as our flexibility:
	// How far can you deviate from the nominal equally spaced schedule?
	// If you like leaky buckets, think about it as the size of your bucket.
	delayVariationTolerance time.Duration
	// Think of the emission interval as the time between events
	// in the nominal equally spaced schedule. If you like leaky buckets,
	// think of it as how frequently the bucket leaks one unit.
	emissionInterval time.Duration

	store store.GCRAStore
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
	if quota.Count <= 0 {
		return nil, fmt.Errorf("Invalid RateQuota %#v. Count must be greater that zero.", quota)
	}
	if quota.Period <= 0 {
		return nil, fmt.Errorf("Invalid RateQuota %#v. Period must be greater that zero.", quota)
	}

	ei := quota.Period / time.Duration(quota.Count)
	if quota.Period%time.Duration(quota.Count) > (quota.Period / 100) {
		return nil, fmt.Errorf("Invalid RateQuota %#v. "+
			"Integer division of Period / Count has a remainder >1%% of the Period. "+
			"This will lead to inaccurate results. Try choosing a larger Period or one "+
			"that is more evenly divisible by the Count.", quota)
	}

	return &gcraLimiter{
		delayVariationTolerance: quota.Period - ei,
		emissionInterval:        ei,
		limit:                   quota.Count,
		store:                   st,
	}, nil
}

func (g *gcraLimiter) Limit(key string) (LimitResult, error) {
	var tat, newTat, now time.Time
	var ttl time.Duration
	limited := false

	i := 0
	for {
		var err error
		var tatVal int64
		var updated bool

		// tat refers to the theoretical arrival time that would be expected
		// from equally spaced requests at exactly the rate limit.
		tatVal, now, err = g.store.GetWithTime(key)
		if err != nil {
			return nil, err
		}

		if tatVal == -1 {
			newTat = now.Add(g.emissionInterval)
			ttl = newTat.Sub(now)
			updated, err = g.store.SetIfNotExistsWithTTL(key, newTat.UnixNano(), ttl)
		} else {
			tat = time.Unix(0, tatVal)

			// Block the request if the next permitted time is in the future
			if now.Before(tat.Add(-g.delayVariationTolerance)) {
				ttl = tat.Sub(now)
				limited = true
				break
			}

			if now.After(tat) {
				newTat = now.Add(g.emissionInterval)
			} else {
				newTat = tat.Add(g.emissionInterval)
			}

			ttl = newTat.Sub(now)
			updated, err = g.store.CompareAndSwapWithTTL(key, tat.UnixNano(), newTat.UnixNano(), ttl)
		}

		if err != nil {
			return nil, err
		}
		if updated {
			break
		}

		i++
		if i > maxCASAttempts {
			return nil, fmt.Errorf(
				"Failed to store updated rate limit data for key %s after %d attempts",
				key, i,
			)
		}
	}

	next := g.delayVariationTolerance - ttl
	var remaining int
	if next < 0 {
		remaining = 0
	} else {
		remaining = int(next/g.emissionInterval + 1)
	}
	retryAfter := -next
	if !limited {
		retryAfter = -1
	}

	return &rateLimitResult{
		limitResult: limitResult{limited},
		limit:       g.limit,
		remaining:   remaining,
		reset:       ttl,
		retryAfter:  retryAfter,
	}, nil
}

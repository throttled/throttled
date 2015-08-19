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

// A RateLimiter manages limiting the rate of actions by key
type RateLimiter interface {
	// RateLimit checks whether a particular key has exceeded a rate
	// limit. It also returns a RateLimitContext to provide additional
	// information about the state of the RateLimiter.
	//
	// If the rate limit has not been exceeded, the underlying storage
	// is updated by the supplied quantity. For example, a quantity of
	// 1 might be used to rate limit a single request while a greater
	// quantity could rate limit based on the size of a file upload in
	// megabytes. If quantity is 0, no update is performed allowing
	// you to "peek" at the state of the RateLimiter for a given key.
	RateLimit(key string, quantity int) (bool, RateLimitContext, error)
}

// RateLimitContext represents the state of the RateLimiter for a
// given key at the time of the query. This state can be used, for
// example, to communicate information to the client via HTTP
// headers. Negative values indicate that the attribute is not
// relevant to the implementation or state.
type RateLimitContext struct {
	// Limit is the maximum number of requests that could be permitted
	// instantaneously for this key starting from an empty state. For
	// example, if a rate limiter allows 10 requests per second per
	// key, Limit would always be 10.
	Limit int

	// Remaining is the maximum number of requests that could be
	// permitted instantaneously for this key given the current
	// state. For example, if a rate limiter allows 10 requests per
	// second and has already received 6 requests for this key this
	// second, Remaining would be 4.
	Remaining int

	// ResetAfter is the time until the RateLimiter returns to its
	// initial state for a given key. For example, if a rate limiter
	// manages requests per second and received one request 200ms ago,
	// Reset would return 800ms. You can also think of this as the time
	// until Limit and Remaining will be equal.
	ResetAfter time.Duration

	// RetryAfter is the time until the next request will be permitted.
	// It should be -1 unless the rate limit has been exceeded.
	RetryAfter time.Duration
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

// GCRARateLimiter is a RateLimiter that users the generic cell-rate
// algorithm.
type GCRARateLimiter struct {
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

// NewGCRARateLimiter creates a GCRARateLimiter. quota.Count defines
// the maximum number of requests permitted in an instantaneous burst
// and quota.Count / quota.Period defines the maximum sustained
// rate. For example, PerMin(60) permits 60 requests instantly per key
// followed by one request per second indefinitely whereas PerSec(1)
// only permits one request per second with no tolerance for bursts.
func NewGCRARateLimiter(st store.GCRAStore, quota RateQuota) (*GCRARateLimiter, error) {
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

	return &GCRARateLimiter{
		delayVariationTolerance: quota.Period - ei,
		emissionInterval:        ei,
		limit:                   quota.Count,
		store:                   st,
	}, nil
}

// RateLimit checks whether a particular key has exceeded a rate
// limit. It also returns a RateLimitContext to provide additional
// information about the state of the RateLimiter.
//
// If the rate limit has not been exceeded, the underlying storage is
// updated by the supplied quantity. For example, a quantity of 1
// might be used to rate limit a single request while a greater
// quantity could rate limit based on the size of a file upload in
// megabytes. If quantity is 0, no update is performed allowing you
// to "peek" at the state of the RateLimiter for a given key.
func (g *GCRARateLimiter) RateLimit(key string, quantity int) (bool, RateLimitContext, error) {
	var tat, newTat, now time.Time
	var ttl time.Duration
	rlc := RateLimitContext{Limit: g.limit}
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
			return false, rlc, err
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
				newTat = now.Add(time.Duration(quantity) * g.emissionInterval)
			} else {
				newTat = tat.Add(time.Duration(quantity) * g.emissionInterval)
			}

			ttl = newTat.Sub(now)
			updated, err = g.store.CompareAndSwapWithTTL(key, tat.UnixNano(), newTat.UnixNano(), ttl)
		}

		if err != nil {
			return false, rlc, err
		}
		if updated {
			break
		}

		i++
		if i > maxCASAttempts {
			return false, rlc, fmt.Errorf(
				"Failed to store updated rate limit data for key %s after %d attempts",
				key, i,
			)
		}
	}

	next := g.delayVariationTolerance - ttl
	if next >= 0 {
		rlc.Remaining = int(next/g.emissionInterval + 1)
	}
	rlc.ResetAfter = ttl
	if limited {
		rlc.RetryAfter = -next
	} else {
		rlc.RetryAfter = -1
	}

	return limited, rlc, nil
}

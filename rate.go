package throttled

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/PuerkitoBio/throttled/store"
)

// Static check to ensure that rateLimiter implements Limiter.
var _ Limiter = (*rateLimiter)(nil)

const (
	maxCASAttempts = 1024
)

// ErrStoreFailed indicates that the throttling function was unable to
// record the request in the store. The caller may choose to
// ignore this error in order to fail open.
var ErrStoreFailed = fmt.Errorf("throttled: Failed to store throttling counter")

// RateLimit creates a throttler that limits the number of requests allowed
// in a certain time window defined by the Quota q. The q parameter specifies
// the requests per time window, and it is silently set to at least 1 request
// and at least a 1 second window if it is less than that.
//
// The vary parameter indicates what criteria should be used to group requests
// for which the limit must be applied (ex.: rate limit based on the remote address).
// See varyby.go for the various options.
//
// The specified store is used to keep track of the rate of requests for each
// key relative to the limit. The throttled package comes with some stores
// in the throttled/store package. Custom stores can be created too, by implementing
// the Store interface.
//
// Requests that bust the rate limit are denied access and go through the denied handler,
// which may be specified on the Throttler and that defaults to the package-global
// variable DefaultDeniedHandler.
//
// The rate limit throttler sets the following headers on the response:
//
//    X-RateLimit-Limit: number of requests allowed in a given burst
//    X-RateLimit-Remaining: instant number of requests permitted before hitting a rate limit
//    X-RateLimit-Reset: seconds before rate limit is fully reset
//
// Additionally, if the request was denied access, the following header is added:
//
//    Retry-After : seconds before the caller should retry
//
func RateLimit(q Quota, vary *VaryBy, store store.Store) *Throttler {
	// Extract requests and window
	reqs, win := q.Quota()

	if reqs < 1 {
		reqs = 1
	}
	if win < time.Second {
		win = time.Second
	}

	emissionInterval := win / time.Duration(reqs)

	// Create and return the throttler
	return &Throttler{
		limiter: &rateLimiter{
			reqs: reqs,
			delayVariationTolerance: win - emissionInterval,
			emissionInterval:        emissionInterval,
			vary:                    vary,
			store:                   store,
			clock:                   func() int64 { return time.Now().UnixNano() },
		},
	}
}

// The rate limiter implements limiting the request to a certain quota
// based on the vary-by criteria. State is saved in the store.
//
// Implementation based on virtual scheduling algorithm from:
// http://en.wikipedia.org/wiki/Generic_cell_rate_algorithm
type rateLimiter struct {
	reqs int
	// Think of the DVT as our flexibility:
	// How far can you deviate from the nominal equally spaced schedule?
	// If you like leaky buckets, think about it as the size of your bucket.
	delayVariationTolerance time.Duration
	// Think of the emission interval as the time between events
	// in the nominal equally spaced schedule. If you like leaky buckets,
	// think of it as how frequently the bucket leaks one unit.
	emissionInterval time.Duration

	vary  *VaryBy
	store store.Store

	clock func() int64
}

// Start initializes the limiter for execution.
func (r *rateLimiter) Start() {}

// Limit is called for each request to the throttled handler. It checks if
// the request can go through and signals it via the returned channel.
// It returns an error if the operation fails.
func (r *rateLimiter) Limit(w http.ResponseWriter, req *http.Request) (<-chan bool, error) {
	// Create return channel and initialize
	ch := make(chan bool, 1)
	// How long until the next request is permitted? Allow the request IFF
	// that is less than zero (in the past).
	var tat, newTat, now int64
	ok := true
	key := r.vary.Key(req)

	i := 0
	for {
		var err error

		now = r.clock()

		// tat refers to the theoretical arrival time that would be expected
		// from equally spaced requests at exactly the rate limit.
		tat, err = r.store.Get(key)
		if err == store.ErrNoSuchKey {
			tat = now
		} else if err != nil {
			return nil, err
		}

		// How long until the next request is permitted? Allow the request
		// IFF that is less the zero (in the past).
		if now < (tat - int64(r.delayVariationTolerance)) {
			ok = false
			break
		}

		if now > tat {
			newTat = now + int64(r.emissionInterval)
		} else {
			newTat = tat + int64(r.emissionInterval)
		}

		var updated bool
		if err == store.ErrNoSuchKey {
			updated, err = r.store.SetNX(key, newTat)
		} else {
			updated, err = r.store.CompareAndSwap(key, tat, newTat)
		}

		if err != nil {
			return nil, err
		}
		if updated {
			tat = newTat
			break
		}

		i++
		if i > maxCASAttempts {
			return nil, ErrStoreFailed
		}
	}

	reset := time.Duration(tat - now)
	next := r.delayVariationTolerance - reset
	remaining := int(next/r.emissionInterval + 1)
	if remaining < 0 {
		remaining = 0
	}

	// Set rate-limit headers
	w.Header().Add("X-RateLimit-Limit", strconv.Itoa(r.reqs))
	w.Header().Add("X-RateLimit-Remaining", strconv.Itoa(remaining))
	w.Header().Add("X-RateLimit-Reset", strconv.Itoa(int(math.Ceil(reset.Seconds()))))
	if !ok {
		w.Header().Add("Retry-After", strconv.Itoa(int(-next.Seconds())))
	}

	// Send response via the return channel
	ch <- ok
	return ch, nil
}

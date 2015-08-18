package throttled

import (
	"math"
	"net/http"
	"strconv"
)

var (
	// DefaultDeniedHandler is the default DeniedHandler for an HTTPLimiter.
	// It returns a 429 status code with a generic message.
	DefaultDeniedHandler = http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "limit exceeded", 429)
	}))

	// DefaultError is the default Error function for an HTTPLimiter.
	// It returns a 500 status code with a generic message.
	DefaultError = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
)

// The LimitResult interface is returned by a Limiter to indicate whether
// a particular request excedes a limit. The concrete type underlying
// the LimitResult might also implement RateLimitResult to expose more detailed
// state.
type LimitResult interface {
	// Limited returns true when a particular request exceded a limit
	Limited() bool
}

// A Limiter manages the limiting of actions by key.
type Limiter interface {
	Limit(key string) (LimitResult, error)
}

// HTTPLimiter faciliates using a Limiter to limit HTTP requests.
type HTTPLimiter struct {
	// DeniedHandler is called if the request is disallowed. If it is nil,
	// the DefaultDeniedHandler variable is used.
	DeniedHandler http.Handler

	// Error is called if the Limiter returns an error. If it is nil,
	// the DefaultErrorFunc is used.
	Error func(w http.ResponseWriter, r *http.Request, err error)

	// Limiter is call for each request to determine whether the request
	// is permitted and update internal state. If it is nil, all requests
	// are permitted.
	Limiter Limiter

	// VaryBy is called for each request to generate a key for the limiter.
	// If it is nil, all requests use an empty string key.
	VaryBy interface {
		Key(*http.Request) string
	}
}

// Limit wraps an http.Handler to limit incoming requests.
// Requests that are not limited will be passed to the handler unchanged.
// Limited requests will be passed to the DeniedHandler.
// If the Limiter returns a RateLimitResult `X-RateLimit-Limit`,
// `X-RateLimit-Remaining`, `X-RateLimit-Reset` and
// `Retry-After` headers will be written to the response.
func (t *HTTPLimiter) Limit(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if t.Limiter == nil {
			h.ServeHTTP(w, r)
			return
		}

		var k string
		if t.VaryBy != nil {
			k = t.VaryBy.Key(r)
		}

		lr, err := t.Limiter.Limit(k)

		if err != nil {
			e := t.Error
			if e == nil {
				e = DefaultError
			}
			e(w, r, err)
			return
		}

		if rlr, ok := lr.(RateLimitResult); ok {
			setRateLimitHeaders(w, rlr)
		}

		if !lr.Limited() {
			h.ServeHTTP(w, r)
			return
		}

		dh := t.DeniedHandler
		if dh == nil {
			dh = DefaultDeniedHandler
		}
		dh.ServeHTTP(w, r)
	})
}

func setRateLimitHeaders(w http.ResponseWriter, rlr RateLimitResult) {
	if v := rlr.Limit(); v >= 0 {
		w.Header().Add("X-RateLimit-Limit", strconv.Itoa(v))
	}

	if v := rlr.Remaining(); v >= 0 {
		w.Header().Add("X-RateLimit-Remaining", strconv.Itoa(v))
	}

	if v := rlr.Reset(); v >= 0 {
		vi := int(math.Ceil(v.Seconds()))
		w.Header().Add("X-RateLimit-Reset", strconv.Itoa(vi))
	}

	if v := rlr.RetryAfter(); v >= 0 {
		vi := int(math.Ceil(v.Seconds()))
		w.Header().Add("Retry-After", strconv.Itoa(vi))
	}
}

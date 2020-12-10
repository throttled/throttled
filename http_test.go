package throttled_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/throttled/throttled/v2"
)

type stubLimiter struct {
}

func (sl *stubLimiter) RateLimitCtx(_ context.Context, key string, quantity int) (bool, throttled.RateLimitResult, error) {
	switch key {
	case "limit":
		result := throttled.RateLimitResult{
			Limit:      -1,
			Remaining:  -1,
			ResetAfter: -1,
			RetryAfter: time.Minute,
		}
		return true, result, nil
	case "error":
		result := throttled.RateLimitResult{}
		return false, result, errors.New("stubLimiter error")
	default:
		result := throttled.RateLimitResult{
			Limit:      1,
			Remaining:  2,
			ResetAfter: time.Minute,
			RetryAfter: -1,
		}
		return false, result, nil
	}
}

type pathGetter struct{}

func (*pathGetter) Key(r *http.Request) string {
	return r.URL.Path
}

type httpTestCase struct {
	path    string
	code    int
	headers map[string]string
}

func TestHTTPRateLimiter(t *testing.T) {
	limiter := throttled.HTTPRateLimiterCtx{
		RateLimiter: &stubLimiter{},
		VaryBy:      &pathGetter{},
	}

	handler := limiter.RateLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	runHTTPTestCases(t, handler, []httpTestCase{
		{"ok", 200, map[string]string{"X-Ratelimit-Limit": "1", "X-Ratelimit-Remaining": "2", "X-Ratelimit-Reset": "60"}},
		{"error", 500, map[string]string{}},
		{"limit", 429, map[string]string{"Retry-After": "60"}},
	})
}

func TestCustomHTTPRateLimiterHandlers(t *testing.T) {
	limiter := throttled.HTTPRateLimiterCtx{
		RateLimiter: &stubLimiter{},
		VaryBy:      &pathGetter{},
		DeniedHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "custom limit exceeded", 400)
		}),
		Error: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, "custom internal error", 501)
		},
	}

	handler := limiter.RateLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	runHTTPTestCases(t, handler, []httpTestCase{
		{"limit", 400, map[string]string{}},
		{"error", 501, map[string]string{}},
	})
}

func runHTTPTestCases(t *testing.T, h http.Handler, cs []httpTestCase) {
	for i, c := range cs {
		req, err := http.NewRequest("GET", c.path, nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if have, want := rr.Code, c.code; have != want {
			t.Errorf("Expected request %d at %s to return %d but got %d",
				i, c.path, want, have)
		}

		for name, want := range c.headers {
			if have := rr.HeaderMap.Get(name); have != want {
				t.Errorf("Expected request %d at %s to have header '%s: %s' but got '%s'",
					i, c.path, name, want, have)
			}
		}
	}
}

func BenchmarkHTTPRateLimiter(b *testing.B) {
	limiter := throttled.HTTPRateLimiterCtx{
		RateLimiter: &stubLimiter{},
		VaryBy:      &pathGetter{},
	}
	h := limiter.RateLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		b.Fatal(err)
	}
	w := httptest.NewRecorder()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.ServeHTTP(w, r)
	}
	_ = w.Body
}

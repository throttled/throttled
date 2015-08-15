package throttled

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type stubLimiter struct {
}

func (sl *stubLimiter) Limit(key string) (LimitResult, error) {
	switch key {
	case "limit":
		return &limitResult{true}, nil
	case "rate":
		return &rateLimitResult{limitResult{false}, 1, 2, time.Minute, -1}, nil
	case "ratelimit":
		return &rateLimitResult{limitResult{true}, -1, -1, -1, time.Minute}, nil
	case "error":
		return nil, errors.New("stubLimiter error")
	default:
		return &limitResult{false}, nil
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

func TestHTTPLimiter(t *testing.T) {
	limiter := HTTPLimiter{
		Limiter: &stubLimiter{},
		VaryBy:  &pathGetter{},
	}

	handler := limiter.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	runHTTPTestCases(t, handler, []httpTestCase{
		{"ok", 200, map[string]string{}},
		{"limit", 429, map[string]string{}},
		{"error", 500, map[string]string{}},
		{"rate", 200, map[string]string{"X-Ratelimit-Limit": "1", "X-Ratelimit-Remaining": "2", "X-Ratelimit-Reset": "60"}},
		{"ratelimit", 429, map[string]string{"Retry-After": "60"}},
	})
}

func TestCustomHTTPLimiterHandlers(t *testing.T) {
	limiter := HTTPLimiter{
		Limiter: &stubLimiter{},
		VaryBy:  &pathGetter{},
		DeniedHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "custom limit exceeded", 400)
		}),
		Error: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, "custom internal error", 501)
		},
	}

	handler := limiter.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

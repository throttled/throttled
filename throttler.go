package throttled

import (
	"net/http"
	"sync"
)

var (
	DefaultDroppedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limit exceeded", http.StatusServiceUnavailable)
	})

	DefaultErrorHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	})
)

type Limiter interface {
	Start()
	Request(http.ResponseWriter, *http.Request) (<-chan bool, error)
}

func Custom(l Limiter) *Throttler {
	return &Throttler{
		limiter: l,
	}
}

type Throttler struct {
	// DroppedHandler is called if the request is disallowed. If it is nil,
	// the DefaultDroppedHandler variable is used.
	DroppedHandler http.Handler

	// ErrorHandler is called if the Allower returns an error. If it is nil,
	// the DefaultErrorHandler variable is used.
	ErrorHandler http.Handler

	limiter Limiter
	mu      sync.Mutex
	started bool
}

func (t *Throttler) Throttle(h http.Handler) http.Handler {
	droph, errh := t.start()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ch, err := t.limiter.Request(w, r)
		if err != nil {
			errh.ServeHTTP(w, r)
			return
		}
		ok := <-ch
		if ok {
			h.ServeHTTP(w, r)
		} else {
			droph.ServeHTTP(w, r)
		}
	})
}

func (t *Throttler) start() (http.Handler, http.Handler) {
	t.mu.Lock()
	defer t.mu.Unlock()
	// Get the effective dropped and error handler
	drop, err := t.DroppedHandler, t.ErrorHandler
	if drop == nil {
		drop = DefaultDroppedHandler
	}
	if err == nil {
		err = DefaultErrorHandler
	}
	if !t.started {
		t.limiter.Start()
		t.started = true
	}
	return drop, err
}

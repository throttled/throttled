package throttled

import (
	"net/http"
	"sync"
)

var (
	DefaultDroppedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limit exceeded", http.StatusServiceUnavailable)
	})

	OnError = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
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

	limiter Limiter
	mu      sync.Mutex
	started bool
}

func (t *Throttler) Throttle(h http.Handler) http.Handler {
	droph := t.start()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ch, err := t.limiter.Request(w, r)
		if err != nil {
			OnError(w, r, err)
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

func (t *Throttler) start() http.Handler {
	t.mu.Lock()
	defer t.mu.Unlock()
	// Get the effective dropped handler
	drop := t.DroppedHandler
	if drop == nil {
		drop = DefaultDroppedHandler
	}
	if !t.started {
		t.limiter.Start()
		t.started = true
	}
	return drop
}

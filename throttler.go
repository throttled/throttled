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

type Throttler struct {
	// DroppedHandler is called if the request is disallowed. If it is nil,
	// the DefaultDroppedHandler variable is used.
	DroppedHandler http.Handler

	// ErrorHandler is called if the Allower returns an error. If it is nil,
	// the DefaultErrorHandler variable is used.
	ErrorHandler http.Handler

	limiter Limiter
	wg      sync.WaitGroup
	stop    chan struct{}
	end     <-chan struct{}

	mu      sync.Mutex
	started bool
}

func (t *Throttler) Throttle(h http.Handler) http.Handler {
	droph, errh := t.start()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.wg.Add(1)
		defer t.wg.Done()
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

// TODO : Close is broken. limiter.process may never see the closed semaphore.
func (t *Throttler) Close() {
	// Make sure no new calls get through
	close(t.stop)
	// Wait for end signal of t.limiter
	<-t.end
	// Wait for goroutines to complete
	t.wg.Wait()
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
		t.stop = make(chan struct{})
		t.end = t.limiter.Start(t.stop)
		t.started = true
	}
	return drop, err
}

// +build ignore

package throttled

import (
	"net/http"
	"sync"
	"time"
)

type Throttle struct {
	delay  time.Duration
	bursts int

	wg      sync.WaitGroup
	bucket  chan chan bool
	stop    chan struct{}
	mu      sync.Mutex
	started bool
	muproc  sync.Mutex
}

// New creates a new throttle using the specified frequency, and allowing
// bursts number of exceeding calls.
func New(freq Delayer, bursts int) *Throttle {
	return &Throttle{delay: freq.Delay(), bursts: bursts}
}

// Func wraps a function and returns a throttled function.
func (t *Throttle) Func(fn func()) func() {
	return t.FuncDropped(fn, func() {})
}

func (t *Throttle) FuncDropped(fn func(), droppedfn func()) func() {
	t.start()
	return func() {
		t.wg.Add(1)
		defer t.wg.Done()
		ch := t.try()
		ok := <-ch
		if ok {
			fn()
		} else {
			droppedfn()
		}
	}
}

func (t *Throttle) Handler(h http.Handler) http.Handler {
	return t.HandlerDropped(h, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Request limit exceeded", http.StatusServiceUnavailable)
	}))
}

func (t *Throttle) HandlerDropped(h http.Handler, droppedh http.Handler) http.Handler {
	t.start()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.wg.Add(1)
		defer t.wg.Done()
		ch := t.try()
		ok := <-ch
		if ok {
			h.ServeHTTP(w, r)
		} else {
			droppedh.ServeHTTP(w, r)
		}
	})
}

// Close the throttle, draining the pending calls. The function
// returns once all pending calls have been processed.
func (t *Throttle) Close() {
	// Make sure no new calls get through
	close(t.stop)
	// Wait for pending calls (drain)
	t.wg.Wait()
	// Safely close the bucket, so that t.process can exit
	close(t.bucket)
	// Wait for t.process to exit
	t.muproc.Lock()
	t.muproc.Unlock()
}

// Prepare the throttle and start processing calls. This function
// is idempotent.
func (t *Throttle) start() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.started {
		if t.bursts <= 1 {
			t.bursts = 1
		}
		t.bucket = make(chan chan bool, t.bursts)
		t.stop = make(chan struct{})
		t.started = true
		go t.process()
	}
}

// Attempt to add the call to the bucket, and return the channel
// that indicates if the call can go through or not.
func (t *Throttle) try() <-chan bool {
	ch := make(chan bool, 1)
	// Check if the bucket is closed.
	select {
	case <-t.stop:
		// Draining, does not accept any new calls
		ch <- false
		return ch
	default:
	}
	// Try to enqueue in the bucket.
	select {
	case t.bucket <- ch:
		return ch
	default:
		// No more space in the bucket
		ch <- false
		return ch
	}
}

// Process the pending calls from the bucket.
func (t *Throttle) process() {
	var after <-chan time.Time
	t.muproc.Lock()
	defer t.muproc.Unlock()
	for v := range t.bucket {
		if after != nil {
			<-after
		}
		v <- true
		after = time.After(t.delay)
	}
}

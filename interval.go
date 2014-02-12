package throttled

import (
	"net/http"
	"sync"
	"time"
)

var _ Limiter = (*intervalLimiter)(nil)

type intervalLimiter struct {
	delay  time.Duration
	bursts int

	lock   sync.RWMutex
	stop   <-chan struct{}
	bucket chan chan bool
	ended  chan struct{}
}

func Interval(delay Delayer, bursts int) *Throttler {
	return &Throttler{
		limiter: &intervalLimiter{
			delay:  delay.Delay(),
			bursts: bursts,
		},
	}
}

func (il *intervalLimiter) Start(stop <-chan struct{}) <-chan struct{} {
	if il.bursts < 0 {
		il.bursts = 0
	}
	il.stop = stop
	il.bucket = make(chan chan bool, il.bursts)
	il.ended = make(chan struct{})
	go il.process()
	return il.ended
}

func (il *intervalLimiter) Request(w http.ResponseWriter, r *http.Request) (<-chan bool, error) {
	ch := make(chan bool, 1)
	if il.stopped() {
		ch <- false
		return ch, nil
	}
	// Mutex required to avoid races with close(bucket) in il.process
	il.lock.RLock()
	defer il.lock.RUnlock()
	select {
	case il.bucket <- ch:
		return ch, nil
	default:
		ch <- false
		return ch, nil
	}
}

func (il *intervalLimiter) stopped() bool {
	select {
	case <-il.stop:
		return true
	default:
		return false
	}
}

func (il *intervalLimiter) process() {
	after := time.After(0)
forever:
	for {
		select {
		case <-il.stop:
			break forever
		case v := <-il.bucket:
			<-after
			// Let the request go through
			v <- true
			// Wait the required duration
			after = time.After(il.delay)
		}
	}
	// Drain remaining buckets
	il.lock.Lock()
	defer il.lock.Unlock()
	close(il.bucket)
	for v := range il.bucket {
		v <- false
	}
	// Notify the end of the process goroutine
	close(il.ended)
}

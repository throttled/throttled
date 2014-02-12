package throttled

import (
	"net/http"
	"time"
)

var _ Limiter = (*intervalLimiter)(nil)

type intervalLimiter struct {
	delay  time.Duration
	bursts int

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
	var after <-chan time.Time
	for v := range il.bucket {
		if after != nil {
			<-after
		}
		// Let the request go through
		v <- true
		// Check if we should stop
		if il.stopped() {
			break
		}
		// Wait the required duration
		after = time.After(il.delay)
	}
	// Drain remaining buckets
	close(il.bucket)
	for v := range il.bucket {
		v <- false
	}
	// Notify the end of the process goroutine
	close(il.ended)
}

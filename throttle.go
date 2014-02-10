package throttled

import (
	"sync"
	"time"
)

type Throttle struct {
	delay   time.Duration
	wg      sync.WaitGroup
	bucket  chan chan bool
	stop    chan struct{}
	mu      sync.Mutex
	started bool
}

func New(freq Delayer, bursts int) *Throttle {
	return &Throttle{delay: freq.Delay()}
}

func (t *Throttle) Func(fn func()) func() {
	t.start()
	return func() {
		defer t.wg.Done()
		ch := t.try()
		ok := <-ch
		if ok {
			fn()
		}
	}
}

func (t *Throttle) start() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.started {
		t.bucket = make(chan chan bool, 1)
		t.stop = make(chan struct{})
		t.started = true
		go t.process()
	}
}

func (t *Throttle) try() <-chan bool {
	ch := make(chan bool, 1)
	select {
	case <-t.stop:
		// Draining, does not accept any new calls
		ch <- false
		return ch
	default:
	}
	select {
	case t.bucket <- ch:
		return ch
	default:
		// No more space in the bucket
		ch <- false
		return ch
	}
}

func (t *Throttle) process() {
	var after <-chan time.Time
	for v := range t.bucket {
		if after != nil {
			<-after
		}
		v <- true
		after = time.After(t.delay)
	}
}

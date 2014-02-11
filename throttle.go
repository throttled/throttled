package throttled

import (
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

func New(freq Delayer, bursts int) *Throttle {
	return &Throttle{delay: freq.Delay(), bursts: bursts}
}

func (t *Throttle) Func(fn func()) func() {
	t.start()
	return func() {
		t.wg.Add(1)
		defer t.wg.Done()
		ch := t.try()
		ok := <-ch
		if ok {
			fn()
		}
	}
}

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

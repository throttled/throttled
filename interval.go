package throttled

import (
	"net/http"
	"sync"
	"time"

	"github.com/golang/groupcache/lru"
)

func Interval(delay Delayer, bursts int, vary *VaryBy) *Throttler {
	var l Limiter
	if vary != nil {
		l = &intervalVaryByLimiter{
			delay:  delay.Delay(),
			bursts: bursts,
			vary:   vary,
		}
	} else {
		l = &intervalLimiter{
			delay:  delay.Delay(),
			bursts: bursts,
		}
	}
	return &Throttler{
		limiter: l,
	}
}

var _ Limiter = (*intervalVaryByLimiter)(nil)
var _ Limiter = (*intervalLimiter)(nil)

type intervalLimiter struct {
	delay  time.Duration
	bursts int

	bucket chan chan bool
}

func (il *intervalLimiter) Start() {
	if il.bursts < 0 {
		il.bursts = 0
	}
	il.bucket = make(chan chan bool, il.bursts)
	go process(il.bucket, il.delay)
}

func (il *intervalLimiter) Request(w http.ResponseWriter, r *http.Request) (<-chan bool, error) {
	ch := make(chan bool, 1)
	select {
	case il.bucket <- ch:
		return ch, nil
	default:
		ch <- false
		return ch, nil
	}
}

type intervalVaryByLimiter struct {
	delay  time.Duration
	bursts int
	vary   *VaryBy

	lock sync.RWMutex
	keys *lru.Cache
}

func (il *intervalVaryByLimiter) Start() {
	if il.bursts < 0 {
		il.bursts = 0
	}
	// If varyby is nil, only one key in the cache
	maxKeys := 1
	if il.vary != nil {
		maxKeys = il.vary.MaxKeys
	}
	il.keys = lru.New(maxKeys)
	il.keys.OnEvicted = il.stopProcess
}

func (il *intervalVaryByLimiter) Request(w http.ResponseWriter, r *http.Request) (<-chan bool, error) {
	ch := make(chan bool, 1)
	key := il.vary.Key(r)

	il.lock.RLock()
	item, ok := il.keys.Get(key)
	if !ok {
		// Create the key, bucket, start goroutine
		// First release the read lock and acquire a write lock
		il.lock.RUnlock()
		il.lock.Lock()
		// Create the bucket, add the key
		bucket := make(chan chan bool, il.bursts)
		il.keys.Add(key, bucket)
		// Start the goroutine to process this bucket
		go process(bucket, il.delay)
		item = bucket
		// Release the write lock, acquire the read lock
		il.lock.Unlock()
		il.lock.RLock()
	}
	defer il.lock.RUnlock()
	bucket := item.(chan chan bool)
	select {
	case bucket <- ch:
		return ch, nil
	default:
		ch <- false
		return ch, nil
	}
}

func process(bucket chan chan bool, delay time.Duration) {
	after := time.After(0)
	for v := range bucket {
		<-after
		// Let the request go through
		v <- true
		// Wait the required duration
		after = time.After(delay)
	}
}

func (il *intervalVaryByLimiter) stopProcess(key lru.Key, value interface{}) {
	close(value.(chan chan bool))
}

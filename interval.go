package throttled

import (
	"net/http"
	"sync"
	"time"

	"github.com/golang/groupcache/lru"
)

var _ Limiter = (*intervalLimiter)(nil)

type intervalLimiter struct {
	delay  time.Duration
	bursts int
	vary   *VaryBy

	lock sync.RWMutex
	keys *lru.Cache
}

func Interval(delay Delayer, bursts int, vary *VaryBy) *Throttler {
	return &Throttler{
		limiter: &intervalLimiter{
			delay:  delay.Delay(),
			bursts: bursts,
			vary:   vary,
		},
	}
}

func (il *intervalLimiter) Start() {
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

func (il *intervalLimiter) Request(w http.ResponseWriter, r *http.Request) (<-chan bool, error) {
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
		go il.process(bucket)
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

func (il *intervalLimiter) process(bucket chan chan bool) {
	after := time.After(0)
	for v := range bucket {
		<-after
		// Let the request go through
		v <- true
		// Wait the required duration
		after = time.After(il.delay)
	}
}

func (il *intervalLimiter) stopProcess(key lru.Key, value interface{}) {
	close(value.(chan chan bool))
}

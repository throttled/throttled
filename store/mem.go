package store

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/golang-lru"
)

type memStore struct {
	sync.RWMutex
	keys *lru.Cache
	m    map[string]*int64
}

// NewMemStore sets up and returns an in-memory store. If maxKeys > 0, the number of different keys
// is restricted to the specified amount. In this case, it uses an LRU algorithm to
// evict older keys to make room for newer ones. If a request is made for a key that
// has been evicted, it will be processed as if its count was 0, possibly allowing requests
// that should be denied.
//
// If maxKeys <= 0, there is no limit on the number of keys, which may use an unbounded amount of
// memory depending on the server's load.
//
// The memStore is only for single-process rate-limiting. To share the rate limit state
// among multiple instances of the web server, use a database- or key-value-based
// store.
//
func NewMemStore(maxKeys int) GCRAStore {
	var m *memStore

	if maxKeys > 0 {
		keys, err := lru.New(maxKeys)
		if err != nil {
			// The interface for `NewMemStore` is part of the public interface so
			// adding an error return would be a breaking change so we panic instead.
			// As of this writing, `lru.New` can only return an error if you pass
			// maxKeys <= 0 so this should never occur.
			panic(err)
		}

		m = &memStore{
			keys: keys,
		}
	} else {
		m = &memStore{
			m: make(map[string]*int64),
		}
	}
	return m
}

func (ms *memStore) Get(key string) (int64, error) {
	valP, ok := ms.get(key, false)

	if !ok {
		return -1, nil
	}

	return atomic.LoadInt64(valP), nil
}

func (ms *memStore) SetIfNotExists(key string, value int64, _ time.Duration) (bool, error) {
	_, ok := ms.get(key, false)

	if ok {
		return false, nil
	}

	ms.Lock()
	defer ms.Unlock()

	_, ok = ms.get(key, true)

	if ok {
		return false, nil
	}

	// Store a pointer to a new instance so that the caller
	// can't mutate the value after setting
	v := value

	if ms.keys != nil {
		ms.keys.Add(key, &v)
	} else {
		ms.m[key] = &v
	}

	return true, nil
}

func (ms *memStore) CompareAndSwap(key string, old, new int64, _ time.Duration) (bool, error) {
	valP, ok := ms.get(key, false)

	if !ok {
		return false, nil
	}

	return atomic.CompareAndSwapInt64(valP, old, new), nil
}

func (ms *memStore) get(key string, locked bool) (*int64, bool) {
	var valP *int64
	var ok bool

	if ms.keys != nil {
		var valI interface{}

		valI, ok = ms.keys.Get(key)
		if ok {
			valP = valI.(*int64)
		}
	} else {
		if !locked {
			ms.RLock()
			defer ms.RUnlock()
		}
		valP, ok = ms.m[key]
	}

	return valP, ok
}

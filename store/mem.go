package store

import (
	"sync"
	"time"

	"github.com/PuerkitoBio/throttled"
)

var _ throttled.StoreTs = (*memStore)(nil)

// memStore implements an in-memory Store.
type memStore struct {
	sync.Mutex
	m map[string]*counter
}

// NewMemStore creates a new MemStore.
func NewMemStore() throttled.Store {
	return &memStore{
		m: make(map[string]*counter),
	}
}

// A counter represents a single entry in the MemStore.
type counter struct {
	n  int
	ts time.Time
}

// GetTs gets a counter from the memory store for the specified key.
// It returns the count and the UTC timestamp, or an error. It returns
// ErrNoSuchKey if the key does not exist in the store.
func (ms *memStore) GetTs(key string) (int, time.Time, error) {
	ms.Lock()
	defer ms.Unlock()
	c := ms.m[key]
	if c == nil {
		return 0, time.Time{}, throttled.ErrNoSuchKey
	}
	return c.n, c.ts, nil
}

// Incr increments the counter for the specified key. It returns the new
// count value, or an error.
func (ms *memStore) Incr(key string) (int, error) {
	ms.Lock()
	defer ms.Unlock()
	c := ms.m[key]
	if c == nil {
		ms.m[key] = &counter{1, time.Now().UTC()}
		return 1, nil
	}
	c.n++
	return c.n, nil
}

// Reset resets the counter for the specified key. It sets the count
// to 1 and initializes the timestamp with the current time, in UTC.
// It returns an error if the operation fails.
func (ms *memStore) Reset(key string, win time.Duration) error {
	ms.Lock()
	defer ms.Unlock()
	c := ms.m[key]
	if c == nil {
		ms.m[key] = &counter{1, time.Now().UTC()}
		return nil
	}
	c.n, c.ts = 1, time.Now().UTC()
	return nil
}

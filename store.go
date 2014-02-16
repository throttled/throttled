package throttled

import (
	"errors"
	"sync"
	"time"
)

// The error returned if the key does not exist in the Store.
var ErrNoSuchKey = errors.New("throttled: no such key")

// Store is the interface to implement to store the RateLimit data.
// Either StoreTs or StoreSecs must be implemented, Store being the
// common base interface.
type Store interface {
	// Incr increments the count for the specified key and returns the new value. It may return an error
	// if the operation fails.
	Incr(string) (int, error)

	// Reset resets the key to 1 with the specified window duration. It
	// returns an error if it fails.
	Reset(string, time.Duration) error
}

// StoreTs extends the Store interface with a getter that returns the count
// and the timestamp (in UTC), or an error.
type StoreTs interface {
	Store

	// Get returns the current request count and the timestamp for the
	// specified key, or an error.
	//
	// The timestamp must be a UTC time.
	GetTs(string) (cnt int, ts time.Time, e error)
}

// StoreSecs extends the Store interface with a getter that returns the count
// and the number of seconds remaining in the current window, or an error.
type StoreSecs interface {
	Store

	// Get returns the current request count and the number of seconds
	// remaining for the specified key, or an error.
	GetSecs(string) (cnt int, secs int, e error)
}

// MemStore implements an in-memory Store.
type MemStore struct {
	sync.Mutex
	m map[string]*counter
}

// NewMemStore creates a new MemStore.
func NewMemStore() *MemStore {
	return &MemStore{
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
func (ms *MemStore) GetTs(key string) (int, time.Time, error) {
	ms.Lock()
	defer ms.Unlock()
	c := ms.m[key]
	if c == nil {
		return 0, time.Time{}, ErrNoSuchKey
	}
	return c.n, c.ts, nil
}

// Incr increments the counter for the specified key. It returns the new
// count value, or an error.
func (ms *MemStore) Incr(key string) (int, error) {
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
func (ms *MemStore) Reset(key string, win time.Duration) error {
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

package throttled

import (
	"sync"
	"time"
)

// TODO : API to be determined, what to store, how to increment atomically, etc.
type Store interface {
	// Incr increments the count for the specified key and returns the new value. It may return an error
	// if the operation fails.
	Incr(string) (int, error)

	// Reset resets the key to 1 with the specified window duration. It
	// returns an error if it fails.
	Reset(string, time.Duration) error
}

type StoreTs interface {
	Store

	// Get returns the current request count and the timestamp for the
	// specified key, or an error.
	//
	// The timestamp must be a UTC time.
	GetTs(string) (cnt int, ts time.Time, e error)
}

type StoreSecs interface {
	Store

	// Get returns the current request count and the number of seconds
	// remaining for the specified key, or an error.
	GetSecs(string) (cnt int, secs float64, e error)
}

type MemStore struct {
	sync.Mutex
	m map[string]*counter
}

func NewMemStore() *MemStore {
	return &MemStore{
		m: make(map[string]*counter),
	}
}

type counter struct {
	n  int
	ts time.Time
}

func (ms *MemStore) GetTs(key string) (int, time.Time, error) {
	ms.Lock()
	defer ms.Unlock()
	c := ms.m[key]
	if c == nil {
		return 0, time.Time{}, nil
	}
	return c.n, c.ts, nil
}

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

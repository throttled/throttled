package store

import (
	"errors"
	"time"
)

// The error returned if the key does not exist in the Store during a
// CompareAndSwap.
var ErrNoSuchKey = errors.New("throttled: no such key")

// Store is the interface to implement to store the RateLimit state (number
// of requests per key, time-to-live or creation timestamp).
type Store interface {
	// Get returns the value of the key if it is in the Store or ErrNoSuchKey
	Get(key string) (int64, error)

	// SetNX sets the value of key only if it is not already set in the Store
	// it returns whether a new value was set.
	SetNX(key string, value int64) (bool, error)

	// CompareAndSwap atomically compares the value at key to the old value.
	// If it matches, it sets it to the new value and returns true. Otherwise,
	// it returns false.
	CompareAndSwap(key string, old, new int64) (bool, error)
}

// RemainingSeconds is a helper function that returns the number of seconds
// remaining from an absolute timestamp in UTC.
func RemainingSeconds(ts time.Time, window time.Duration) int {
	return int((window - time.Now().UTC().Sub(ts)).Seconds())
}

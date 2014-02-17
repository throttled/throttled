package throttled

import (
	"errors"
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

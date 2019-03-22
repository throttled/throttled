// Package radixstore offers Redis-based store implementation for throttled
// using radix.
package radixstore // import "github.com/throttled/throttled/store/radixstore"

import (
	"strconv"
	"strings"
	"time"

	"github.com/mediocregopher/radix"
)

const (
	redisCASMissingKey = "key does not exist"
	redisCASScript     = `
local v = redis.call('get', KEYS[1])
if v == false then
  return redis.error_reply("key does not exist")
end
if v ~= ARGV[1] then
  return 0 
end
redis.call('setex', KEYS[1], ARGV[3], ARGV[2])
return 1
`
)

// RadixStore implements a Redis-based store using radix.
type RadixStore struct {
	pool   *radix.Pool
	prefix string
}

// New creates a new Redis-based store using the provided pool to get its
// connections. The keys will have the specified keyPrefix, which may be an
// empty string. Any updating operations will reset the key TTL to the provided
// value rounded down to the nearest second. Depends on Redis 2.6+ for EVAL
// support.
func New(pool *radix.Pool, keyPrefix string) (*RadixStore, error) {
	return &RadixStore{
		pool:   pool,
		prefix: keyPrefix,
	}, nil
}

// GetWithTime returns the value of the key if it is in the store or -1 if it
// does not exist. It also returns the current time at the redis server to
// microsecond precision.
func (r *RadixStore) GetWithTime(key string) (int64, time.Time, error) {
	var now time.Time

	key = r.prefix + key

	var t []int64
	var v int64
	mn := radix.MaybeNil{Rcv: &v}
	p := radix.Pipeline(
		radix.Cmd(&t, "TIME"),
		radix.Cmd(&mn, "GET", key),
	)
	if err := r.pool.Do(p); err != nil {
		return 0, now, err
	}

	if t == nil || len(t) != 2 {
		return -1, now, nil
	}
	now = time.Unix(t[0], t[1]*int64(time.Microsecond))

	if mn.Nil {
		return -1, now, nil
	}

	return v, now, nil
}

// SetIfNotExistsWithTTL sets the value of key only if it is not already set in
// the store. It returns whether a new value was set. If a new value was set,
// the ttl in the key is also set—though this operation is not performed
// atomically.
func (r *RadixStore) SetIfNotExistsWithTTL(
	key string, value int64, ttl time.Duration,
) (bool, error) {
	key = r.prefix + key

	var v int64
	if err := r.pool.Do(radix.FlatCmd(&v, "SETNX", key, value)); err != nil {
		return false, err
	}

	updated := v == 1

	ttlSeconds := int(ttl.Seconds())

	// An `EXPIRE 0` will delete the key immediately, so make sure that we set
	// expiry for a minimum of one second out so that our results stay in the
	// store.
	if ttlSeconds < 1 {
		ttlSeconds = 1
	}

	if err := r.pool.Do(
		radix.FlatCmd(nil, "EXPIRE", key, ttlSeconds),
	); err != nil {
		return updated, err
	}

	return updated, nil
}

// CompareAndSwapWithTTL atomically compares the value at key to the old value.
// If it matches, it sets it to the new value and returns true. Otherwise, it
// returns false. If the key does not exist in the store, it returns false with
// no error. If the swap succeeds, the ttl for the key is updated atomically.
func (r *RadixStore) CompareAndSwapWithTTL(
	key string, old int64, new int64, ttl time.Duration,
) (bool, error) {
	key = r.prefix + key

	ttlSeconds := int(ttl.Seconds())

	// An `EXPIRE 0` will delete the key immediately, so make sure that we set
	// expiry for a minimum of one second out so that our results stay in the
	// store.
	if ttlSeconds < 1 {
		ttlSeconds = 1
	}

	script := radix.NewEvalScript(1, redisCASScript)
	args := []string{
		// Keys
		key,
		// Args
		strconv.FormatInt(old, 10),
		strconv.FormatInt(new, 10),
		strconv.Itoa(ttlSeconds),
	}

	var swapped bool
	if err := r.pool.Do(script.Cmd(&swapped, args...)); err != nil {
		if strings.Contains(err.Error(), redisCASMissingKey) {
			return false, nil
		}

		return false, err
	}

	return swapped, nil
}

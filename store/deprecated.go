// Package store contains deprecated aliases for subpackages
package store // import "github.com/throttled/throttled/v2/store"

import (
	"github.com/gomodule/redigo/redis"
	"github.com/throttled/throttled/v2"
	"github.com/throttled/throttled/v2/store/memstore"
	"github.com/throttled/throttled/v2/store/redigostore"
)

// NewMemStore initializes a new memory-based store.
//
// Deprecated: Use github.com/throttled/throttled/v2/store/memstore instead.
func NewMemStore(maxKeys int) throttled.Store {
	st, err := memstore.New(maxKeys)
	if err != nil {
		// As of this writing, `lru.New` can only return an error if you pass
		// maxKeys <= 0 so this should never occur.
		panic(err)
	}
	return st
}

// NewRedisStore initializes a new Redigo-based store.
//
// Deprecated: Use github.com/throttled/throttled/v2/store/redigostore instead.
func NewRedisStore(pool *redis.Pool, keyPrefix string, db int) throttled.Store {
	st, err := redigostore.New(pool, keyPrefix, db)
	if err != nil {
		// As of this writing, creating a Redis store never returns an error
		// so this should be safe while providing some ability to return errors
		// in the future.
		panic(err)
	}
	return st
}

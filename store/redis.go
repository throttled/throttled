package store

import (
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
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
if ARGV[3] ~= "0" then
  redis.call('setex', KEYS[1], ARGV[3], ARGV[2])
else
  redis.call('set', KEYS[1], ARGV[2])
end
return 1
`
)

// redisStore implements a Redis-based store.
type redisStore struct {
	pool         *redis.Pool
	prefix       string
	db           int
	ttl          time.Duration
	supportsEval bool
}

// NewRedisStore creates a new Redis-based store, using the provided pool to get its
// connections. The keys will have the specified keyPrefix, which may be an empty string,
// and the database index specified by db will be selected to store the keys. Any
// updating operations will reset the key TTL to the provided value rounded down to
// the nearest second.
func NewRedisStore(pool *redis.Pool, keyPrefix string, db int, ttl time.Duration) Store {
	return &redisStore{
		pool:         pool,
		prefix:       keyPrefix,
		db:           db,
		ttl:          ttl,
		supportsEval: true,
	}
}

func (r *redisStore) Get(key string) (int64, error) {
	key = r.prefix + key

	conn, err := r.getConn()
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	v, err := redis.Int64(conn.Do("GET", key))
	if err == redis.ErrNil {
		return 0, ErrNoSuchKey
	} else if err != nil {
		return 0, err
	}

	return v, nil
}

func (r *redisStore) SetNX(key string, value int64) (bool, error) {
	key = r.prefix + key

	conn, err := r.getConn()
	if err != nil {
		return false, err
	}
	defer conn.Close()

	v, err := redis.Int64(conn.Do("SETNX", key, value))
	if err != nil {
		return false, err
	}

	updated := v == 1

	if r.ttl > 0 {
		if _, err := conn.Do("EXPIRE", key, int(r.ttl.Seconds())); err != nil {
			return updated, err
		}
	}

	return updated, nil
}

func (r *redisStore) CompareAndSwap(key string, old, new int64) (bool, error) {
	key = r.prefix + key
	conn, err := r.getConn()
	if err != nil {
		return false, err
	}
	defer conn.Close()

	if r.supportsEval {
		swapped, err := r.compareAndSwapWithEval(conn, key, old, new)
		if err == nil {
			return swapped, nil
		}

		// If failure is due to EVAL being unsupported, note that and
		// retry using WATCH
		if strings.Contains(err.Error(), "unknown command") {
			r.supportsEval = false
		} else {
			return false, err
		}
	}

	swapped, err := r.compareAndSwapWithWatch(conn, key, old, new)
	if err != nil {
		return false, err
	}

	return swapped, nil
}

func (r *redisStore) compareAndSwapWithWatch(conn redis.Conn, key string, old, new int64) (bool, error) {
	conn.Send("WATCH", key)
	conn.Send("GET", key)
	conn.Flush()
	conn.Receive()

	v, err := redis.Int64(conn.Receive())
	if err == redis.ErrNil {
		return false, ErrNoSuchKey
	}
	if v != old {
		return false, nil
	}

	conn.Send("MULTI")
	if r.ttl > 0 {
		conn.Send("SETEX", key, int(r.ttl.Seconds()), new)
	} else {
		conn.Send("SET", key, new)
	}
	if _, err := conn.Do("EXEC"); err == redis.ErrNil {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

func (r *redisStore) compareAndSwapWithEval(conn redis.Conn, key string, old, new int64) (bool, error) {
	swapped, err := redis.Bool(conn.Do("EVAL", redisCASScript, 1, key, old, new, int(r.ttl.Seconds())))
	if err != nil {
		if strings.Contains(err.Error(), redisCASMissingKey) {
			err = ErrNoSuchKey
		}

		return false, err
	}

	return swapped, nil
}

// Select the specified database index.
func (r *redisStore) getConn() (redis.Conn, error) {
	conn := r.pool.Get()

	// Select the specified database
	if r.db > 0 {
		if _, err := redis.String(conn.Do("SELECT", r.db)); err != nil {
			conn.Close()
			return nil, err
		}
	}

	return conn, nil
}

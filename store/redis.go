package store

import (
	"time"

	"github.com/PuerkitoBio/throttled"
	"github.com/garyburd/redigo/redis"
)

type redisStore struct {
	pool   *redis.Pool
	prefix string
	db     int
}

func NewRedisStore(pool *redis.Pool, keyPrefix string, db int) throttled.Store {
	return &redisStore{
		pool:   pool,
		prefix: keyPrefix,
		db:     db,
	}
}

func (r *redisStore) Incr(key string, window time.Duration) (int, int, error) {
	conn := r.pool.Get()
	defer conn.Close()
	if _, err := redis.String(conn.Do("SELECT", r.db)); err != nil {
		return 0, 0, err
	}
	conn.Send("MULTI")
	conn.Send("INCR", r.prefix+key)
	conn.Send("TTL", r.prefix+key)
	vals, err := redis.Values(conn.Do("EXEC"))
	if err != nil {
		conn.Do("DISCARD")
		return 0, 0, err
	}
	var cnt, ttl int
	if _, err = redis.Scan(vals, &cnt, &ttl); err != nil {
		return 0, 0, err
	}
	if ttl == -1 {
		ttl = int(window.Seconds())
		_, err = conn.Do("EXPIRE", r.prefix+key, ttl)
		if err != nil {
			return 0, 0, err
		}
	}
	return cnt, ttl, nil
}

// Reset sets the value of the key to 1, and resets its time window.
func (r *redisStore) Reset(key string, window time.Duration) error {
	conn := r.pool.Get()
	defer conn.Close()
	if _, err := redis.String(conn.Do("SELECT", r.db)); err != nil {
		return err
	}
	_, err := redis.String(conn.Do("SET", r.prefix+key, "1", "EX", int(window.Seconds()), "NX"))
	return err
}

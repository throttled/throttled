package store

import (
	"strconv"
	"time"

	"github.com/PuerkitoBio/throttled"
	"github.com/garyburd/redigo/redis"
)

var _ throttled.StoreSecs = (*redisStore)(nil)

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

func (r *redisStore) Incr(key string) (int, error) {
	conn := r.pool.Get()
	defer conn.Close()
	if _, err := redis.String(conn.Do("SELECT", r.db)); err != nil {
		return 0, err
	}
	return redis.Int(conn.Do("INCR", r.prefix+key))
}

func (r *redisStore) Reset(key string, window time.Duration) error {
	conn := r.pool.Get()
	defer conn.Close()
	if _, err := redis.String(conn.Do("SELECT", r.db)); err != nil {
		return err
	}
	_, err := redis.String(conn.Do("SETEX", r.prefix+key, int(window.Seconds()), 1))
	return err
}

func (r *redisStore) GetSecs(key string) (int, int, error) {
	conn := r.pool.Get()
	defer conn.Close()
	if _, err := redis.String(conn.Do("SELECT", r.db)); err != nil {
		return 0, 0, err
	}
	conn.Send("MULTI")
	conn.Send("GET", r.prefix+key)
	conn.Send("TTL", r.prefix+key)
	vals, err := redis.Values(conn.Do("EXEC"))
	if err != nil {
		conn.Do("DISCARD")
		return 0, 0, err
	}
	var scnt string
	var cnt, secs int
	if _, err = redis.Scan(vals, &scnt, &secs); err != nil {
		return 0, 0, err
	}
	if scnt == "" {
		return 0, 0, throttled.ErrNoSuchKey
	}
	if cnt, err = strconv.Atoi(scnt); err != nil {
		return 0, 0, err
	}
	return cnt, secs, nil
}

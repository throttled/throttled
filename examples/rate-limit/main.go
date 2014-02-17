package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/PuerkitoBio/throttled"
	"github.com/PuerkitoBio/throttled/store"
	"github.com/garyburd/redigo/redis"
)

var (
	requests  = flag.Int("requests", 10, "number of requests allowed in the time window")
	window    = flag.Duration("window", time.Minute, "time window for the limit of requests")
	storeType = flag.String("store", "mem", "store to use, one of `mem`, `redis` (on default localhost port) or `memcached`")
	delayRes  = flag.Duration("delay-response", 0, "delay the response by a random duration between 0 and this value")
	quiet     = flag.Bool("quiet", false, "close to no output")
)

func main() {
	flag.Parse()

	var h http.Handler
	var ok, ko int
	var mu sync.Mutex
	var st throttled.Store

	start := time.Now()
	switch *storeType {
	case "mem":
		st = store.NewMemStore(0)
	case "redis":
		st = store.NewRedisStore(setupRedis(), "throttled:", 0)
	case "memcached":
	default:
		log.Fatalf("unsupported store: %s", *storeType)
	}
	t := throttled.RateLimit(throttled.CustomQuota{*requests, *window}, &throttled.VaryBy{
		Path: true,
	}, st)
	t.DroppedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !*quiet {
			log.Printf("KO: %s", time.Since(start))
		}
		throttled.DefaultDroppedHandler(w, r)
		mu.Lock()
		defer mu.Unlock()
		ko++
	})
	rand.Seed(time.Now().Unix())
	h = t.Throttle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !*quiet {
			log.Printf("ok: %s", time.Since(start))
		}
		if *delayRes > 0 {
			wait := time.Duration(rand.Intn(int(*delayRes)))
			time.Sleep(wait)
		}
		w.WriteHeader(200)
		mu.Lock()
		defer mu.Unlock()
		ok++
	}))

	// Print stats once in a while
	go func() {
		for _ = range time.Tick(10 * time.Second) {
			mu.Lock()
			log.Printf("ok: %d, ko: %d", ok, ko)
			mu.Unlock()
		}
	}()
	fmt.Println("server listening on port 9000")
	http.ListenAndServe(":9000", h)
}

func setupRedis() *redis.Pool {
	pool := &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 30 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", ":6379")
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
	return pool
}

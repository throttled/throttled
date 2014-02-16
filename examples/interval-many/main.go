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
)

var (
	delay    = flag.Duration("delay", 200*time.Millisecond, "delay between calls")
	bursts   = flag.Int("bursts", 10, "number of bursts allowed")
	delayRes = flag.Duration("delay-response", 0, "delay the response by a random duration between 0 and this value")
	quiet    = flag.Bool("quiet", false, "close to no output")
)

func main() {
	flag.Parse()

	var ok, ko int
	var mu sync.Mutex

	start := time.Now()
	t := throttled.Interval(throttled.Delay(*delay), *bursts, nil)
	t.DroppedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !*quiet {
			log.Printf("%s: KO: %s", r.URL.Path, time.Since(start))
		}
		w.WriteHeader(503)
		mu.Lock()
		defer mu.Unlock()
		ko++
	})
	rand.Seed(time.Now().Unix())
	makeHandler := func(ix int) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !*quiet {
				log.Printf("handler %d: %s: ok: %s", ix, r.URL.Path, time.Since(start))
			}
			if *delayRes > 0 {
				wait := time.Duration(rand.Intn(int(*delayRes)))
				time.Sleep(wait)
			}
			w.WriteHeader(200)
			mu.Lock()
			defer mu.Unlock()
			ok++
		})
	}
	h1 := t.Throttle(makeHandler(1))
	h2 := t.Throttle(makeHandler(2))
	mux := http.NewServeMux()
	mux.Handle("/a", h1)
	mux.Handle("/b", h2)

	// Print stats once in a while
	go func() {
		for _ = range time.Tick(10 * time.Second) {
			mu.Lock()
			log.Printf("ok: %d, ko: %d", ok, ko)
			mu.Unlock()
		}
	}()
	fmt.Println("server listening on port 9000")
	http.ListenAndServe(":9000", mux)
}

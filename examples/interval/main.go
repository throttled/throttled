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
	requests = flag.Int("requests", 20, "number of web requests to launch")
	delay    = flag.Duration("delay", 200*time.Millisecond, "delay between calls")
	bursts   = flag.Int("bursts", 10, "number of bursts allowed")
	server   = flag.Bool("server-only", false, "run the server only")
)

type delayer time.Duration

func (d delayer) Delay() time.Duration {
	return time.Duration(d)
}

func main() {
	flag.Parse()

	var h http.Handler
	var ok, ko int
	var mu sync.Mutex

	start := time.Now()
	t := throttled.Interval(delayer(*delay), *bursts)
	t.DroppedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("web: KO: %s", time.Since(start))
		w.WriteHeader(503)
		mu.Lock()
		defer mu.Unlock()
		ko++
	})
	h = t.Throttle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("web: ok: %s", time.Since(start))
		w.WriteHeader(200)
		mu.Lock()
		defer mu.Unlock()
		ok++
	}))
	if *server {
		fmt.Println("server listening on port 9000")
		http.ListenAndServe(":9000", h)
		return
	}
	go http.ListenAndServe(":9000", h)
	fmt.Println("server listening on port 9000")
	rand.Seed(time.Now().UTC().UnixNano())
	for i := 0; i < *requests; i++ {
		go func() {
			_, err := http.Get("http://localhost:9000/")
			if err != nil {
				fmt.Println("error: ", err)
			}
		}()
		wait := rand.Intn(100) + 1
		<-time.After(time.Duration(wait) * time.Millisecond)
	}
	time.Sleep((*delay) * time.Duration(*requests))
	t.Close()
	mu.Lock()
	defer mu.Unlock()
	fmt.Printf("\nok: %d / KO: %d", ok, ko)
}

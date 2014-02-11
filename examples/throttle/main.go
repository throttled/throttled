package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/PuerkitoBio/throttled"
)

var (
	funcs    = flag.Int("funcs", 10, "number of functions to launch")
	requests = flag.Int("requests", 10, "number of web requests to launch")
	delay    = flag.Duration("delay", 200*time.Millisecond, "delay between calls")
	bursts   = flag.Int("bursts", 10, "number of bursts allowed")
)

type delayer time.Duration

func (d delayer) Delay() time.Duration {
	return time.Duration(d)
}

func main() {
	flag.Parse()

	var fn func()
	var h http.Handler

	start := time.Now()
	t := throttled.New(delayer(*delay), *bursts)
	if *funcs > 0 {
		fn = t.FuncDropped(func() {
			fmt.Printf("fn: ok: %s\n", time.Since(start))
		}, func() {
			fmt.Printf("fn: ko: %s\n", time.Since(start))
		})
	}
	if *requests > 0 {
		h = t.HandlerDropped(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("web: ok: %s\n", time.Since(start))
			w.WriteHeader(200)
		}), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("web: ko: %s\n", time.Since(start))
			w.WriteHeader(503)
		}))
	}
	if h != nil {
		go http.ListenAndServe(":9000", h)
		fmt.Println("server listening on port 9000")
	}
	var (
		cntfn int
		cnth  int
	)
	rand.Seed(time.Now().UTC().UnixNano())
	for i := 0; i < (*funcs)+(*requests); i++ {
		r := rand.Intn(2)
		if cnth == *requests {
			r = 0
		}
		switch r {
		case 0:
			if cntfn < *funcs {
				go fn()
				cntfn++
				break
			}
			fallthrough
		default:
			go func() {
				_, err := http.Get("http://localhost:9000/")
				if err != nil {
					fmt.Println("error: ", err)
				}
			}()
		}
		wait := rand.Intn(100) + 1
		<-time.After(time.Duration(wait) * time.Millisecond)
	}
	time.Sleep((*delay) * time.Duration(*funcs+*requests))
}

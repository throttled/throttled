package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/PuerkitoBio/throttled"
)

var (
	numgc    = flag.Int("gc", 0, "number of GC runs")
	mallocs  = flag.Int("mallocs", 0, "number of mallocs")
	total    = flag.Int("total", 0, "total number of bytes allocated")
	allocs   = flag.Int("allocs", 0, "number of bytes allocated")
	refrate  = flag.Duration("refresh", 0, "refresh rate of the memory stats")
	delayRes = flag.Duration("delay-response", 0, "delay the response by a random duration between 0 and this value")
	quiet    = flag.Bool("quiet", false, "close to no output")
)

func main() {
	flag.Parse()

	var h http.Handler
	var ok, ko int
	var mu sync.Mutex
	var mem runtime.MemStats

	start := time.Now()
	runtime.ReadMemStats(&mem)
	thresholds := &runtime.MemStats{}
	if *numgc > 0 {
		thresholds.NumGC = mem.NumGC + uint32(*numgc)
	}
	if *mallocs > 0 {
		thresholds.Mallocs = mem.Mallocs + uint64(*mallocs)
	}
	if *total > 0 {
		thresholds.TotalAlloc = mem.TotalAlloc + uint64(*total)
	}
	if *allocs > 0 {
		thresholds.Alloc = mem.Alloc + uint64(*allocs)
	}
	t := throttled.MemStats(thresholds, *refrate)
	t.DroppedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !*quiet {
			log.Printf("web: KO: %s", time.Since(start))
		}
		w.WriteHeader(503)
		mu.Lock()
		defer mu.Unlock()
		ko++
	})
	rand.Seed(time.Now().Unix())
	h = t.Throttle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !*quiet {
			log.Printf("web: ok: %s", time.Since(start))
		}
		if *delayRes > 0 {
			wait := time.Duration(rand.Intn(int(*delayRes)))
			time.Sleep(wait)
		}
		// Read the whole file in memory, to actually use 64Kb (instead of streaming to w)
		b, err := ioutil.ReadFile("test-file")
		if err != nil {
			throttled.OnError(w, r, err)
			return
		}
		_, err = w.Write(b)
		if err != nil {
			throttled.OnError(w, r, err)
		}
		mu.Lock()
		defer mu.Unlock()
		ok++
	}))

	// Print stats once in a while
	go func() {
		var mem runtime.MemStats
		for _ = range time.Tick(10 * time.Second) {
			mu.Lock()
			runtime.ReadMemStats(&mem)
			log.Printf("ok: %d, ko: %d", ok, ko)
			log.Printf("TotalAllocs: %d Kb, Allocs: %d Kb, Mallocs: %d, NumGC: %d", mem.TotalAlloc/1024, mem.Alloc/1024, mem.Mallocs, mem.NumGC)
			mu.Unlock()
		}
	}()
	fmt.Println("server listening on port 9000")
	http.ListenAndServe(":9000", h)
}

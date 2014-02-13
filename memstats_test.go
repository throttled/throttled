package throttled

import (
	"fmt"
	"net/http"
	"runtime"
	"testing"
	"time"
)

func TestMemStatsGC(t *testing.T) {
	var ms runtime.MemStats
	// Configure the throttler
	ith := Interval(PerSec(5), 10)
	st := &stats{body: func() {
		runtime.GC()
	}}
	runtime.ReadMemStats(&ms)
	thresholds := &runtime.MemStats{NumGC: ms.NumGC + 2}
	th := MemStats(thresholds, 0, 10)
	// Use interval to control calls one after another
	h := ith.Throttle(th.Throttle(st))
	runTestHandler(h, 5, time.Second)
	// Assert the results
	ok, _, _ := st.Stats()
	if ok > 2 {
		t.Errorf("NumGC: expected at most 2 calls, got %d", ok)
	}
}

func TestMemStatsAlloc(t *testing.T) {
	var ms runtime.MemStats
	var escape *[]byte
	// Configure the throttler
	ith := Interval(PerSec(100), 1000)
	st := &stats{body: func() {
		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)
		fmt.Printf("before: %d\n", mem.TotalAlloc)
		b := make([]byte, 100000)
		escape = &b
		fmt.Printf("after: %d\n", mem.TotalAlloc)
	}}
	runtime.ReadMemStats(&ms)
	thresholds := &runtime.MemStats{TotalAlloc: ms.TotalAlloc + 300000}
	th := MemStats(thresholds, 100*time.Millisecond, 100)
	th.DroppedHandler = http.HandlerFunc(st.DroppedHTTP)
	h := ith.Throttle(th.Throttle(st))
	runTestHandler(h, 100, 5*time.Second)
	// Assert the results
	ok, dropped, _ := st.Stats()
	if ok < 2 || ok > 4 {
		t.Errorf("TotalAlloc: expected between 2 and 10 calls, got %d", ok)
	}
	if dropped != 100-ok {
		t.Errorf("TotalAlloc: expected %d dropped, got %d", 100-ok, dropped)
	}
}
func BenchmarkReadMemStats(b *testing.B) {
	var mem runtime.MemStats
	for i := 0; i < b.N; i++ {
		runtime.ReadMemStats(&mem)
	}
}

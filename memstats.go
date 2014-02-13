package throttled

import (
	"net/http"
	"runtime"
	"sync"
	"time"
)

var _ Limiter = (*memStatsLimiter)(nil)

type memStatsLimiter struct {
	thresholds  *runtime.MemStats
	refreshRate time.Duration

	lockStats sync.RWMutex
	stats     runtime.MemStats

	lockBucket sync.RWMutex
	stop       <-chan struct{}
	bucket     chan chan bool
	ended      chan struct{}
}

func MemStats(thresholds *runtime.MemStats, refreshRate time.Duration, buffer int) *Throttler {
	return &Throttler{
		limiter: &memStatsLimiter{
			thresholds:  thresholds,
			refreshRate: refreshRate,
			bucket:      make(chan chan bool, buffer),
		},
	}
}

func (m *memStatsLimiter) Start(stop <-chan struct{}) <-chan struct{} {
	m.stop = stop
	m.ended = make(chan struct{})
	// Make sure there is an initial MemStats reading
	runtime.ReadMemStats(&m.stats)
	if m.refreshRate > 0 {
		go m.refresh()
	}
	go m.process()
	return m.ended
}

func (m *memStatsLimiter) refresh() {
forever:
	for {
		select {
		case <-m.stop:
			break forever
		case <-time.After(m.refreshRate):
			m.lockStats.Lock()
			runtime.ReadMemStats(&m.stats)
			m.lockStats.Unlock()
		}
	}
}

func (m *memStatsLimiter) Request(w http.ResponseWriter, r *http.Request) (<-chan bool, error) {
	ch := make(chan bool, 1)
	if m.stopped() {
		ch <- false
		return ch, nil
	}
	// Check if memory thresholds are reached
	if !m.allow() {
		ch <- false
		return ch, nil
	}
	m.lockBucket.RLock()
	defer m.lockBucket.RUnlock()
	select {
	case m.bucket <- ch:
		return ch, nil
	default:
		ch <- false
		return ch, nil
	}
}

func (m *memStatsLimiter) allow() bool {
	m.lockStats.RLock()
	defer m.lockStats.RUnlock()
	// If refreshRate == 0, then read on every request.
	mem := m.stats
	if m.refreshRate == 0 {
		runtime.ReadMemStats(&mem)
	}
	ok := true
	checkStat(m.thresholds.Alloc, mem.Alloc, &ok)
	checkStat(m.thresholds.BuckHashSys, mem.BuckHashSys, &ok)
	checkStat(m.thresholds.Frees, mem.Frees, &ok)
	checkStat(m.thresholds.GCSys, mem.GCSys, &ok)
	checkStat(m.thresholds.HeapAlloc, mem.HeapAlloc, &ok)
	checkStat(m.thresholds.HeapIdle, mem.HeapIdle, &ok)
	checkStat(m.thresholds.HeapInuse, mem.HeapInuse, &ok)
	checkStat(m.thresholds.HeapObjects, mem.HeapObjects, &ok)
	checkStat(m.thresholds.HeapReleased, mem.HeapReleased, &ok)
	checkStat(m.thresholds.HeapSys, mem.HeapSys, &ok)
	checkStat(m.thresholds.LastGC, mem.LastGC, &ok)
	checkStat(m.thresholds.Lookups, mem.Lookups, &ok)
	checkStat(m.thresholds.MCacheInuse, mem.MCacheInuse, &ok)
	checkStat(m.thresholds.MCacheSys, mem.MCacheSys, &ok)
	checkStat(m.thresholds.MSpanInuse, mem.MSpanInuse, &ok)
	checkStat(m.thresholds.MSpanSys, mem.MSpanSys, &ok)
	checkStat(m.thresholds.Mallocs, mem.Mallocs, &ok)
	checkStat(m.thresholds.NextGC, mem.NextGC, &ok)
	checkStat(uint64(m.thresholds.NumGC), uint64(mem.NumGC), &ok)
	checkStat(m.thresholds.OtherSys, mem.OtherSys, &ok)
	checkStat(m.thresholds.PauseTotalNs, mem.PauseTotalNs, &ok)
	checkStat(m.thresholds.StackInuse, mem.StackInuse, &ok)
	checkStat(m.thresholds.StackSys, mem.StackSys, &ok)
	checkStat(m.thresholds.Sys, mem.Sys, &ok)
	checkStat(m.thresholds.TotalAlloc, mem.TotalAlloc, &ok)
	return ok
}

func checkStat(threshold, actual uint64, ok *bool) {
	if !*ok {
		return
	}
	if threshold > 0 {
		if actual >= threshold {
			*ok = false
		}
	}
}

func (m *memStatsLimiter) stopped() bool {
	select {
	case <-m.stop:
		return true
	default:
		return false
	}
}

func (m *memStatsLimiter) process() {
forever:
	for {
		select {
		case <-m.stop:
			break forever
		case v := <-m.bucket:
			// Let the request go through
			v <- true
		}
	}
	// Drain remaining buckets
	m.lockBucket.Lock()
	defer m.lockBucket.Unlock()
	close(m.bucket)
	for v := range m.bucket {
		v <- false
	}
	// Notify the end of the process goroutine
	close(m.ended)
}

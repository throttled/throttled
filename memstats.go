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
	stats     *runtime.MemStats

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
	go m.process()
	go m.refresh()
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
			runtime.ReadMemStats(m.stats)
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
	if m.thresholds.Alloc > 0 {
		if m.stats.Alloc >= m.thresholds.Alloc {
			return false
		}
	}
	if m.thresholds.BuckHashSys > 0 {
		if m.stats.BuckHashSys >= m.thresholds.BuckHashSys {
			return false
		}
	}
	return true
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

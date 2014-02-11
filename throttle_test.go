package throttled

import (
	"sync"
	"testing"
	"time"
)

func TestThrottleFuncCall(t *testing.T) {
	ok := false
	th := New(PerSec(1), 0)
	fn := th.Func(func() {
		ok = true
	})
	fn()

	if !ok {
		t.Error("expected func to be called")
	}
}

func TestThrottleFunc(t *testing.T) {
	var ts []time.Time
	var cnt int
	var mu sync.Mutex
	n := 5
	th := New(PerSec(4), n-1) // Timestamp should be 250ms apart, one call should be dropped
	fn := th.Func(func() {
		mu.Lock()
		defer mu.Unlock()
		ts = append(ts, time.Now())
		cnt++
	})
	for i := 0; i < n; i++ {
		go fn()
	}
	time.Sleep(time.Second)
	mu.Lock()
	defer mu.Unlock()
	if cnt != n-1 {
		t.Errorf("expected %d calls, got %d", n-1, cnt)
	}
	for i := 0; i < cnt-1; i++ {
		if (ts[i+1].Sub(ts[i]) < 250*time.Millisecond) || (ts[i+1].Sub(ts[i]) > 270*time.Millisecond) {
			t.Errorf("expected calls to be 250ms apart")
		}
	}
}

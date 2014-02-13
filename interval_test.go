package throttled

import (
	"testing"
	"time"
)

func TestInterval(t *testing.T) {
	cases := []struct {
		rate     Delayer
		bursts   int
		handlers int
		min      int
		max      int
	}{
		0: {PerSec(50), 3, 6, 3, 5},
		1: {PerSec(20), 0, 1, 1, 1},
		2: {PerSec(30), 0, 3, 1, 2},
		3: {PerSec(4), 2, 6, 2, 4},
		4: {PerSec(0), 0, 6, 1, 6},
		5: {PerSec(0), 10, 6, 6, 6},
	}
	for i, c := range cases {
		// Configure the throttler
		th := Interval(c.rate, c.bursts)
		// Run the tests
		st, resps := runTest(th, c.handlers, 100*time.Millisecond+(c.rate.Delay()*time.Duration(c.handlers)), nil)
		// Assert the results
		ok, dropped, ts := st.Stats()
		// Test that the number of OK calls are within min and max
		if ok < c.min || ok > c.max {
			t.Errorf("%d: expected between %d and %d calls, got %d", i, c.min, c.max, ok)
		}
		// The number of dropped calls should balance
		if expdrop := (c.handlers - ok); dropped != expdrop {
			t.Errorf("%d: expected %d dropped, got %d", i, expdrop, dropped)
		}
		// Test that the timestamps are separated by the rate's delay
		for j := 0; j < len(ts)-1; j++ {
			if (ts[j+1].Sub(ts[j]) < c.rate.Delay()) || (ts[j+1].Sub(ts[j]) > c.rate.Delay()+50*time.Millisecond) {
				t.Errorf("%d: expected calls to be %s apart", i, c.rate.Delay())
			}
		}
		// Test that the right status codes have been received
		twos, fives := 0, 0
		for j := 0; j < len(resps); j++ {
			if resps[j].StatusCode == 200 {
				twos++
			} else if resps[j].StatusCode == 503 {
				fives++
			} else {
				t.Errorf("%d: unexpected status code: %d", i, resps[j].StatusCode)
			}
		}
		if twos != st.ok {
			t.Errorf("%d: expected %d status 200, got %d", i, ok, twos)
		}
		if fives != (c.handlers - st.ok) {
			t.Errorf("%d: expected %d status 503, got %d", i, c.handlers-ok, fives)
		}
	}
}

// TODO : Because of the random nature of the select when multiple cases are available,
// very hard to write a test that checks how many will be dropped.
func TestIntervalClose(t *testing.T) {
	t.Skip("unreliable test")
	cases := []struct {
		rate     Delayer
		bursts   int
		handlers int
		wait     time.Duration
		min      int
		max      int
	}{
		0: {PerSec(1), 5, 10, 200 * time.Millisecond, 8, 9},
	}
	for i, c := range cases {
		th := Interval(c.rate, c.bursts)
		// Run the tests
		st, _ := runTest(th, c.handlers, c.wait, nil)
		// Assert the results
		_, dropped, _ := st.Stats()
		if dropped < c.min || dropped > c.max {
			t.Errorf("%d: expected between %d and %d dropped calls, got %d", i, c.min, c.max, dropped)
		}
	}
}

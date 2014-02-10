package throttled

import (
	"testing"
	"time"
)

func TestDelayer(t *testing.T) {
	cases := []struct {
		in  Delayer
		out time.Duration
	}{
		0:  {PerSec(1), time.Second},
		1:  {PerSec(2), 500 * time.Millisecond},
		2:  {PerSec(4), 250 * time.Millisecond},
		3:  {PerSec(5), 200 * time.Millisecond},
		4:  {PerSec(10), 100 * time.Millisecond},
		5:  {PerSec(100), 10 * time.Millisecond},
		6:  {PerSec(3), 333333333 * time.Nanosecond},
		7:  {PerMin(1), time.Minute},
		8:  {PerMin(2), 30 * time.Second},
		9:  {PerMin(4), 15 * time.Second},
		10: {PerMin(5), 12 * time.Second},
		11: {PerMin(10), 6 * time.Second},
		12: {PerMin(60), time.Second},
	}
	for i, c := range cases {
		got := c.in.Delay()
		if got != c.out {
			t.Errorf("%d: expected %s, got %s", i, c.out, got)
		}
	}
}

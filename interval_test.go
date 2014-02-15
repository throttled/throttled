package throttled

import (
	"net/http"
	"testing"

	"github.com/PuerkitoBio/boom/commands"
)

func TestInterval(t *testing.T) {
	cases := []struct {
		n      int
		c      int
		rps    int
		bursts int
	}{
		0: {60, 10, 20, 100},
		1: {300, 20, 100, 100},
		2: {10, 10, 1, 10},
		3: {1000, 100, 1000, 100},
	}
	for i, c := range cases {
		// Setup the stats handler
		st := &stats{}
		// Create the throttler
		th := Interval(PerSec(c.rps), c.bursts, nil)
		th.DroppedHandler = http.HandlerFunc(st.DroppedHTTP)
		b := commands.Boom{
			Req:    &commands.ReqOpts{},
			N:      c.n,
			C:      c.c,
			Output: "sshh",
		}
		// Run the test
		rpt := runTest(th.Throttle(st), b)
		// Assert results
		wigglef := 0.2 * float64(c.rps)
		if rpt.SuccessRPS < float64(c.rps)-wigglef || rpt.SuccessRPS > float64(c.rps)+wigglef {
			t.Errorf("%d: expected RPS to be around %d, got %f", i, c.rps, rpt.SuccessRPS)
		}
		ok, ko, _ := st.Stats()
		if ok != rpt.StatusCodeDist[200] {
			t.Errorf("%d: expected %d status 200, got %d", i, rpt.StatusCodeDist[200], ok)
		}
		if ko != rpt.StatusCodeDist[503] {
			t.Errorf("%d: expected %d status 503, got %d", i, rpt.StatusCodeDist[503], ok)
		}
		if len(rpt.StatusCodeDist) > 2 {
			t.Errorf("%d: expected at most 2 different status codes, got %d", i, len(rpt.StatusCodeDist))
		}
		//interval := PerSec(c.rps).Delay()
		//wiggled := time.Duration(0.2 * float64(interval))
		//for j := 0; j < len(ts)-1; j++ {
		//	gap := ts[j+1].Sub(ts[j])
		//	if gap < interval-wiggled || gap > interval+wiggled {
		//		t.Errorf("%d: expected timestamps to be within %s, got %s", i, interval, gap)
		//	}
		//}
	}
}

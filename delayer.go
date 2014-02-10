package throttled

import "time"

type Delayer interface {
	Delay() time.Duration
}

type PerSec int

func (ps PerSec) Delay() time.Duration {
	if ps <= 0 {
		return 0
	}
	return time.Duration(float64(1.0/float64(ps)) * float64(time.Second))
}

type PerMin int

func (pm PerMin) Delay() time.Duration {
	if pm <= 0 {
		return 0
	}
	return time.Duration(float64(1.0/float64(pm)) * float64(time.Minute))
}

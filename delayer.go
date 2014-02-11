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
	return time.Duration(1.0 / float64(ps) * float64(time.Second))
}

type PerMin int

func (pm PerMin) Delay() time.Duration {
	if pm <= 0 {
		return 0
	}
	return time.Duration(1.0 / float64(pm) * float64(time.Minute))
}

type PerHour int

func (ph PerHour) Delay() time.Duration {
	if ph <= 0 {
		return 0
	}
	return time.Duration(1.0 / float64(ph) * float64(time.Hour))
}

type Delay time.Duration

func (d Delay) Delay() time.Duration {
	return time.Duration(d)
}

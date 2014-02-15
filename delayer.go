package throttled

import "time"

type Quota interface {
	Quota() (int, time.Duration)
}

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

func (ps PerSec) Quota() (int, time.Duration) {
	return int(ps), time.Second
}

type PerMin int

func (pm PerMin) Delay() time.Duration {
	if pm <= 0 {
		return 0
	}
	return time.Duration(1.0 / float64(pm) * float64(time.Minute))
}

func (pm PerMin) Quota() (int, time.Duration) {
	return int(pm), time.Minute
}

type PerHour int

func (ph PerHour) Delay() time.Duration {
	if ph <= 0 {
		return 0
	}
	return time.Duration(1.0 / float64(ph) * float64(time.Hour))
}

func (ph PerHour) Quota() (int, time.Duration) {
	return int(ph), time.Hour
}

type PerDay int

func (pd PerDay) Delay() time.Duration {
	if pd <= 0 {
		return 0
	}
	return time.Duration(1.0 / float64(pd) * float64(24*time.Hour))
}

func (pd PerDay) Quota() (int, time.Duration) {
	return int(pd), 24 * time.Hour
}

type Delay time.Duration

func (d Delay) Delay() time.Duration {
	return time.Duration(d)
}

type CustomQuota struct {
	Requests int
	Window   time.Duration
}

func (c CustomQuota) Quota() (int, time.Duration) {
	return c.Requests, c.Window
}

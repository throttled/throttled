package throttled

import "time"

var DefaultStore Store

// TODO : API to be determined, what to store, how to increment atomically, etc.
type Store interface {
	Init(reqs int, window time.Duration)
	Get(string) (int, int, error)
	Incr(string) error
}

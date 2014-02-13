package throttled

// TODO : API to be determined, what to store, how to increment atomically, etc.

var DefaultStore Store

type Store interface {
	Get(string) (int, error)
	Incr(string, int) error
}

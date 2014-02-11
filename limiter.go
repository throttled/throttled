package throttled

type Limiter interface {
	Start()
	Stop()
	Request() <-chan bool
}

type intervalLimiter struct {
	bucket chan chan bool
	stop   chan struct{}
}

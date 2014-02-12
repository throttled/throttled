package throttled

import "net/http"

type Limiter interface {
	Start(<-chan struct{}) <-chan struct{}
	Request(http.ResponseWriter, *http.Request) (<-chan bool, error)
}

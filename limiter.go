package throttled

import "net/http"

type Limiter interface {
	Start()
	Request(http.ResponseWriter, *http.Request) (<-chan bool, error)
}

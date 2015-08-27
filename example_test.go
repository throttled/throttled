package throttled_test

import (
	"log"
	"net/http"

	"gopkg.in/throttled/throttled.v1"
	"gopkg.in/throttled/throttled.v1/store"
)

var myHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hi there!"))
})

// ExampleHTTPRateLimiter demonstrates the usage of HTTPRateLimiter
// for rate-limiting access to an http.Handler to 20 requests per path
// per minute with a maximum burst of 5 requests.
func ExampleHTTPRateLimiter() {
	st := store.NewMemStore(65536)
	rq := throttled.RateQuota{throttled.PerMin(20), 5}
	rateLimiter, err := throttled.NewGCRARateLimiter(st, rq)
	if err != nil {
		log.Fatal(err)
	}

	httpRateLimiter := throttled.HTTPRateLimiter{
		RateLimiter: rateLimiter,
		VaryBy:      &throttled.VaryBy{Path: true},
	}

	http.ListenAndServe(":8080", httpRateLimiter.RateLimit(myHandler))
}

package throttled_test

import (
	"log"
	"net/http"

	"gopkg.in/throttled/throttled.v1"
	"gopkg.in/throttled/throttled.v1/store/mem"
)

var myHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hi there!"))
})

// ExampleHTTPRateLimiter demonstrates the usage of HTTPRateLimiter
// for rate-limiting access to an http.Handler to 20 requests per path
// per minute with a maximum burst of 5 requests.
func ExampleHTTPRateLimiter() {
	rq := throttled.RateQuota{throttled.PerMin(20), 5}
	st, err := mem.New(65536)
	if err != nil {
		log.Fatal(err)
	}

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

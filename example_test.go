package throttled_test

import (
	"log"
	"net/http"

	"gopkg.in/throttled/throttled.v0"
	"gopkg.in/throttled/throttled.v0/store"
)

var myHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hi there!"))
})

// ExampleHTTPRateLimiter demonstrates the usage of HTTPRateLimiter for rate-limiting
// access to an http.Handler to 5 requests per path per minute.
func ExampleHTTPRateLimiter() {
	st := store.NewMemStore(256)
	limit := throttled.PerMin(5)
	rateLimiter, err := throttled.NewGCRARateLimiter(st, limit)
	if err != nil {
		log.Fatal(err)
	}

	httpRateLimiter := throttled.HTTPRateLimiter{
		RateLimiter: rateLimiter,
		VaryBy:      &throttled.VaryBy{Path: true},
	}

	http.ListenAndServe(":8080", httpRateLimiter.RateLimit(myHandler))
}

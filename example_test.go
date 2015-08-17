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

// ExampleHTTPLimiter demonstrates the usage of HTTPLimiter for rate-limiting
// access to an http.Handler to 5 requests per path per minute.
func ExampleHTTPLimiter() {
	st := store.NewMemStore(256)
	lim := throttled.PerMin(5)
	rateLim, err := throttled.NewGCRARateLimiter(st, lim)
	if err != nil {
		log.Fatal(err)
	}

	httpLim := throttled.HTTPLimiter{
		Limiter: rateLim,
		VaryBy:  &throttled.VaryBy{Path: true},
	}

	http.ListenAndServe(":8080", httpLim.Limit(myHandler))
}

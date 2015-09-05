package throttled_test

import (
	"fmt"
	"log"
	"net/http"

	"gopkg.in/throttled/throttled.v2"
	"gopkg.in/throttled/throttled.v2/store/memstore"
)

var myHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hi there!"))
})

// ExampleHTTPRateLimiter demonstrates the usage of HTTPRateLimiter
// for rate-limiting access to an http.Handler to 20 requests per path
// per minute with a maximum burst of 5 requests.
func ExampleHTTPRateLimiter() {
	store, err := memstore.New(65536)
	if err != nil {
		log.Fatal(err)
	}

	// Maximum burst of 5 which refills at 20 tokens per minute.
	quota := throttled.RateQuota{throttled.PerMin(20), 5}

	rateLimiter, err := throttled.NewGCRARateLimiter(store, quota)
	if err != nil {
		log.Fatal(err)
	}

	httpRateLimiter := throttled.HTTPRateLimiter{
		RateLimiter: rateLimiter,
		VaryBy:      &throttled.VaryBy{Path: true},
	}

	http.ListenAndServe(":8080", httpRateLimiter.RateLimit(myHandler))
}

// Demonstrates direct use of GCRARateLimiter's RateLimit function (and the
// more general RateLimiter interface). This should be used anywhere where
// granular control over rate limiting is required.
func ExampleGCRARateLimiter() {
	store, err := memstore.New(65536)
	if err != nil {
		log.Fatal(err)
	}

	// Maximum burst of 5 which refills at 20 tokens per minute.
	quota := throttled.RateQuota{throttled.PerMin(20), 5}

	rateLimiter, err := throttled.NewGCRARateLimiter(store, quota)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Bucket based on the address of the origin request. This actually
		// includes a remote port as well and as such, is not actually
		// particularly useful for rate limiting, but does serve as a simple
		// demonstration.
		bucket := r.RemoteAddr

		// add one token to the bucket
		limited, result, err := rateLimiter.RateLimit(bucket, 1)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "Internal error: %v.", err.Error())
		}

		if limited {
			w.WriteHeader(429)
			fmt.Fprintf(w, "Rate limit exceeded. Please try again in %v.",
				result.RetryAfter)
		} else {
			fmt.Fprintf(w, "Hello.")
		}
	})
	http.ListenAndServe(":8080", nil)
}

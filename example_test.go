package throttled_test

import (
	"fmt"
	"log"
	"net/http"
	"time"

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

	// Maximum burst of 10 which gets a new token once every 5 minutes.
	quota := throttled.RateQuota{throttled.PerHour(12), 10}

	rateLimiter, err := throttled.NewGCRARateLimiter(store, quota)
	if err != nil {
		log.Fatal(err)
	}

	// Bucket according to the hour of the day (0-23). This has the effect of
	// allowing a new burst of ten requests every hour, and with a consistent
	// refill of a new token every 5 minutes.
	bucket := fmt.Sprintf("per-hour:%v\n", time.Now().Hour())

	limited, result, err := rateLimiter.RateLimit(bucket, 1)
	if limited {
		fmt.Printf("Rate limit exceeded. Please try again in %v.",
			result.RetryAfter)
	} else {
		fmt.Printf("Operation successful.")
	}
}

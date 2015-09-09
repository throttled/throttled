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

	// Maximum burst of 5 which refills at 1 token per hour.
	quota := throttled.RateQuota{throttled.PerHour(1), 5}

	rateLimiter, err := throttled.NewGCRARateLimiter(store, quota)
	if err != nil {
		log.Fatal(err)
	}

	// Bucket according to the our number i / 10 (so 1 falls into the bucket 0
	// while 11 falls into the bucket 1). This has the effect of allowing a
	// burst of 5 plus 1 (a single emission interval) on every ten iterations
	// of the loop. See the output for better clarity here.
	//
	// We also refill the bucket at 1 token per hour, but that has no effect
	// for the purposes of this example.
	for i := 0; i < 20; i++ {
		bucket := fmt.Sprintf("by-order:%v", i/10)

		limited, result, err := rateLimiter.RateLimit(bucket, 1)
		if err != nil {
			log.Fatal(err)
		}

		if limited {
			fmt.Printf("Bucket %v: FAILED. Rate limit exceeded.\n", bucket)
		} else {
			fmt.Printf("Bucket %v: Operation successful (remaining=%v).\n",
			  bucket, result.Remaining)
		}
	}

	// Output:
	// Bucket by-order:0: Operation successful (remaining=5).
	// Bucket by-order:0: Operation successful (remaining=4).
	// Bucket by-order:0: Operation successful (remaining=3).
	// Bucket by-order:0: Operation successful (remaining=2).
	// Bucket by-order:0: Operation successful (remaining=1).
	// Bucket by-order:0: Operation successful (remaining=0).
	// Bucket by-order:0: FAILED. Rate limit exceeded.
	// Bucket by-order:0: FAILED. Rate limit exceeded.
	// Bucket by-order:0: FAILED. Rate limit exceeded.
	// Bucket by-order:0: FAILED. Rate limit exceeded.
	// Bucket by-order:1: Operation successful (remaining=5).
	// Bucket by-order:1: Operation successful (remaining=4).
	// Bucket by-order:1: Operation successful (remaining=3).
	// Bucket by-order:1: Operation successful (remaining=2).
	// Bucket by-order:1: Operation successful (remaining=1).
	// Bucket by-order:1: Operation successful (remaining=0).
	// Bucket by-order:1: FAILED. Rate limit exceeded.
	// Bucket by-order:1: FAILED. Rate limit exceeded.
	// Bucket by-order:1: FAILED. Rate limit exceeded.
	// Bucket by-order:1: FAILED. Rate limit exceeded.
}

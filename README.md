# Throttled [![build status](https://secure.travis-ci.org/throttled/throttled.png)](https://travis-ci.org/throttled/throttled) [![GoDoc](https://godoc.org/gopkg.in/throttled/throttled.v0?status.png)](https://godoc.org/gopkg.in/throttled/throttled.v0)

Package throttled implements rate limiting access to resources such as
HTTP endpoints.

## Installation

`go get gopkg.in/throttled/throttled.v0`

As of July 27, 2015, the package is located under its own Github
organization.  Please adjust your imports to
`gopkg.in/throttled/throttled.v0`.

## Documentation

API documentation is available on [godoc.org][doc]. The following
example demonstrates the usage of HTTPLimiter for rate-limiting access
to an http.Handler to 20 requests per path per minute with bursts of
up to 5 additional requests:

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

## Versioning

throttled uses gopkg.in for semantic versioning. 

The 1.x release series is compatible with the original, unversioned
library written by [Martin Angers][puerkitobio]. There is a
[blog post explaining that version's usage on 0value.com][blog].

## License

The [BSD 3-clause license][bsd]. Copyright (c) 2014 Martin Angers and Contributors.

[blog]: http://0value.com/throttled--guardian-of-the-web-server
[bsd]: https://opensource.org/licenses/BSD-3-Clause
[doc]: https://godoc.org/gopkg.in/throttled/throttled.v0
[puerkitobio]: https://github.com/puerkitobio/

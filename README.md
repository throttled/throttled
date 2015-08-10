# Throttled [![build status](https://secure.travis-ci.org/throttled/throttled.png)](https://travis-ci.org/throttled/throttled) [![GoDoc](https://godoc.org/gopkg.in/throttled/throttled.v0?status.png)](https://godoc.org/gopkg.in/throttled/throttled.v0)

Package throttled implements various strategies for limiting access to resources,
such as for rate limiting HTTP requests.

## Installation

`go get gopkg.in/throttled/throttled.v0`

As of July 27, 2015, the package is located under its own Github organization.
Please adjust your imports to `gopkg.in/throttled/throttled.v0`.

## Documentation

API documentation is available on [godoc.org][doc]. The following example
demonstrates the usage of HTTPLimiter for rate-limiting
access to an http.Handler to 5 requests per path per minute.

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

## Versioning

throttled uses gopkg.in for semantic versioning. 

The 0.x release series is compatible with the original, unversioned library written
by [Martin Angers][puerkitobio]. There is a [blog post explaining that version's usage on 0value.com][blog].

## License

The [BSD 3-clause license][bsd]. Copyright (c) 2014 Martin Angers and Contributors.

[blog]: http://0value.com/throttled--guardian-of-the-web-server
[bsd]: https://opensource.org/licenses/BSD-3-Clause
[doc]: https://godoc.org/gopkg.in/throttled/throttled.v0
[examples]: https://github.com/throttled/throttled/tree/master/examples
[puerkitobio]: https://github.com/puerkitobio/

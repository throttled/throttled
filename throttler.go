package throttled

import (
	"bytes"
	"net/http"
	"strings"
)

var (
	DefaultDropHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limit exceeded", http.StatusServiceUnavailable)
	})

	DefaultStoreErrorHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	})
)

type Throttler interface {
	Throttle(http.Handler) http.Handler
	ThrottleDrop(http.Handler, http.Handler) http.Handler
	Close()
	SetStore(Store)
}

type VaryBy struct {
	RemoteAddr bool
	Method     bool
	Path       bool
	Headers    []string
	Params     []string
	Cookies    []string
	Separator  string
}

func (vb *VaryBy) Key(r *http.Request) string {
	var buf bytes.Buffer

	if vb == nil {
		return "" // Special case for no vary-by option
	}
	sep := vb.Separator
	if sep == "" {
		sep = "\n" // Separator defaults to newline
	}
	if vb.RemoteAddr {
		buf.WriteString(strings.ToLower(r.RemoteAddr) + sep)
	}
	if vb.Method {
		buf.WriteString(strings.ToLower(r.Method) + sep)
	}
	for _, h := range vb.Headers {
		buf.WriteString(strings.ToLower(r.Header.Get(h)) + sep)
	}
	if vb.Path {
		buf.WriteString(r.URL.Path + sep)
	}
	for _, p := range vb.Params {
		buf.WriteString(r.FormValue(p) + sep)
	}
	for _, c := range vb.Cookies {
		ck, err := r.Cookie(c)
		if err == nil {
			buf.WriteString(ck.Value)
		}
		buf.WriteString(sep) // Write the separator anyway, whether or not the cookie exists
	}
	return buf.String()
}

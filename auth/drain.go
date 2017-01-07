package auth

import (
	"net/http"
	"strings"
)

type drainTokenAuth struct {
	handler http.Handler
	tokens  map[string]bool
}

// DrainTokenAuth is an authentication middleware matching a set of tokens against the Logplex-Drain-Token header.
func DrainTokenAuth(tokens string) func(http.Handler) http.Handler {
	set := make(map[string]bool)

	for _, token := range strings.Split(tokens, ",") {
		if token != "" {
			set[token] = true
		}
	}

	fn := func(h http.Handler) http.Handler {
		return &drainTokenAuth{
			handler: h,
			tokens:  set,
		}
	}
	return fn
}

func (d drainTokenAuth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if d.authenticate(r) == false {
		http.Error(w, http.StatusText(401), 401)
		return
	}

	d.handler.ServeHTTP(w, r)
}

func (d drainTokenAuth) authenticate(r *http.Request) bool {
	if len(d.tokens) == 0 {
		return true
	}

	if token := r.Header.Get("Logplex-Drain-Token"); token != "" {
		_, ok := d.tokens[token]
		return ok
	}

	return false
}

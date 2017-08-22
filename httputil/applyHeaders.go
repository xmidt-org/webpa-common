package httputil

import (
	"fmt"
	"net/http"
)

// ApplyHeaders injects static HTTP headers into every http.ResponseWriter that passes
// through the decorated handlers.  The h parameter may be any of the following types:
//
//   http.Header
//   map[string][]string
//   map[string]string
//
// Headers are injected prior to invoking the delegate handler.
//
// If h evaluates to nil or an empty map, the returned constructor will be a noop.
func ApplyHeaders(h interface{}) func(http.Handler) http.Handler {
	var source http.Header

	switch v := h.(type) {
	case nil:
		// fallthrough to the logic below

	case http.Header:
		source = v

	case map[string][]string:
		source = v

	case map[string]string:
		source = make(http.Header, len(v))
		for name, value := range v {
			source[name] = []string{value}
		}

	default:
		panic(fmt.Errorf("Unsupported header type: %T", h))
	}

	if len(source) > 0 {
		return func(delegate http.Handler) http.Handler {
			return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				output := response.Header()
				for k, v := range source {
					output[k] = v
				}

				delegate.ServeHTTP(response, request)
			})
		}
	}

	// if no headers are specified, default to a noop constructor
	return func(delegate http.Handler) http.Handler {
		return delegate
	}
}

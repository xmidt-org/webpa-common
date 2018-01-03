package xhttp

import (
	"net/http"
	"net/textproto"
)

// StaticHeaders returns an Alice-style constructor that emits a static set of headers
// into every response.  If the set of headers is empty, the constructor does no
// decoration.
func StaticHeaders(extra http.Header) func(http.Handler) http.Handler {
	if len(extra) > 0 {
		// preprocess the header keys, so that we do this just once.
		// this also allows the header to be read in from sources that do not use
		// the http.Header methods, such as unmarshaled JSON.
		preprocessed := make(http.Header, len(extra))
		for k, v := range extra {
			preprocessed[textproto.CanonicalMIMEHeaderKey(k)] = v
		}

		extra = preprocessed
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				header := response.Header()
				for k, v := range extra {
					header[k] = v
				}

				next.ServeHTTP(response, request)
			})
		}
	}

	return func(next http.Handler) http.Handler {
		return next
	}
}

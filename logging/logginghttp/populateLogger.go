package logginghttp

import (
	"net/http"

	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/log"
)

var (
	requestProtoKey  interface{} = "requestProto"
	requestMethodKey interface{} = "requestMethod"
	requestURIKey    interface{} = "requestURI"
	remoteAddrKey    interface{} = "remoteAddr"
)

// RequestProtoKey returns the contextual logging key for an HTTP request's protocol
func RequestProtoKey() interface{} {
	return requestProtoKey
}

// RequestMethodKey returns the contextual logging key for an HTTP request's method
func RequestMethodKey() interface{} {
	return requestMethodKey
}

// RequestURIKey returns the contextual logging key for an HTTP request's unmodified URI
func RequestURIKey() interface{} {
	return requestURIKey
}

// RemoteAddr returns the contextual logging key for an HTTP request's remote address,
// as filled in by the enclosing http.Server.
func RemoteAddrKey() interface{} {
	return remoteAddrKey
}

// PopulateLogger produces an Alice-style decorator that emits a decorated go-kit logger into the request context.
// The supplied base go-kit Logger is decorated for each request with information about the request.  Downstream code
// can then use this logger via logging.GetLogger(request.Context()).
//
// If the base parameter is not supplied, the default logger is decorated for each request.
func PopulateLogger(base log.Logger) func(http.Handler) http.Handler {
	if base == nil {
		base = logging.DefaultLogger()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, request *http.Request) {
			ctx := logging.WithLogger(
				request.Context(),
				log.With(
					base,
					requestProtoKey, request.Proto,
					requestMethodKey, request.Method,
					requestURIKey, request.RequestURI,
					remoteAddrKey, request.RemoteAddr,
				),
			)

			next.ServeHTTP(rw, request.WithContext(ctx))
		})
	}
}

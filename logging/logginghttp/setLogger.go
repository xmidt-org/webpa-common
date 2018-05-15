package logginghttp

import (
	"context"
	"net/http"

	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/log"
)

var (
	requestMethodKey interface{} = "requestMethod"
	requestURIKey    interface{} = "requestURI"
	remoteAddrKey    interface{} = "remoteAddr"
)

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

// LoggerFunc is a strategy for adding key/value pairs (possibly) based on an HTTP request.
// Functions of this type must append key/value pairs to the supplied slice and then return
// the new slice.
type LoggerFunc func([]interface{}, *http.Request) []interface{}

// StandardKeyValues is a LoggerFunc that adds the request information described by logging keys in this package.
func StandardKeyValues(kv []interface{}, request *http.Request) []interface{} {
	return append(kv,
		requestMethodKey, request.Method,
		requestURIKey, request.RequestURI,
		remoteAddrKey, request.RemoteAddr,
	)

}

// SetLogger produces a go-kit RequestFunc that inserts a go-kit Logger into the context.
// Zero or more LoggerFuncs can be provided to added key/values.  Note that nothing is added to
// the base logger by default.  If no LoggerFuncs are supplied, the base Logger is added to the
// context as is.  In particular, StandardKeyValues must be used to inject the request method, uri, etc.
//
// The base logger must be non-nil.  There is no default applied.
//
// The returned function can be used with xcontext.Populate.
func SetLogger(base log.Logger, lf ...LoggerFunc) func(context.Context, *http.Request) context.Context {
	if base == nil {
		panic("The base Logger cannot be nil")
	}

	if len(lf) > 0 {
		return func(ctx context.Context, request *http.Request) context.Context {
			kv := []interface{}{}
			for _, f := range lf {
				kv = f(kv, request)
			}

			return logging.WithLogger(
				ctx,
				log.With(base, kv...),
			)
		}
	}

	return func(ctx context.Context, _ *http.Request) context.Context {
		return logging.WithLogger(ctx, base)
	}
}

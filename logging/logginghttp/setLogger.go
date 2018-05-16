package logginghttp

import (
	"context"
	"net/http"
	"net/textproto"

	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/log"
	"github.com/gorilla/mux"
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

// RequestInfo is a LoggerFunc that adds the request information described by logging keys in this package.
func RequestInfo(kv []interface{}, request *http.Request) []interface{} {
	return append(kv,
		requestMethodKey, request.Method,
		requestURIKey, request.RequestURI,
		remoteAddrKey, request.RemoteAddr,
	)

}

// Header returns a logger func that extracts the value of a header and inserts it as the
// value of a logging key.  If the header is not present in the request, a blank string
// is set as the logging key's value.
func Header(headerName, keyName string) LoggerFunc {
	headerName = textproto.CanonicalMIMEHeaderKey(headerName)

	return func(kv []interface{}, request *http.Request) []interface{} {
		values := request.Header[headerName]
		switch len(values) {
		case 0:
			return append(kv, keyName, "")
		case 1:
			return append(kv, keyName, values[0])
		default:
			return append(kv, keyName, values)
		}
	}
}

// PathVariable returns a LoggerFunc that extracts the value of a gorilla/mux path variable and inserts
// it into the value of a logging key.  If the variable is not present, a blank string is
// set as the logging key's value.
func PathVariable(variableName, keyName string) LoggerFunc {
	return func(kv []interface{}, request *http.Request) []interface{} {
		variables := mux.Vars(request)
		if len(variables) > 0 {
			return append(kv, keyName, variables[variableName])
		}

		return append(kv, keyName, "")
	}
}

// SetLogger produces a go-kit RequestFunc that inserts a go-kit Logger into the context.
// Zero or more LoggerFuncs can be provided to added key/values.  Note that nothing is added to
// the base logger by default.  If no LoggerFuncs are supplied, the base Logger is added to the
// context as is.  In particular, RequestInfo must be used to inject the request method, uri, etc.
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

package fanout

import (
	"context"
	"net/http"
	"net/textproto"
	"net/url"

	"github.com/Comcast/webpa-common/xhttp"
)

// Endpoints is a strategy interface for determining the set of HTTP URL endpoints that a fanout
// should use.  Each returned endpoint will be associated with a single http.Request object and transaction.
type Endpoints interface {
	NewEndpoints(*http.Request) ([]*url.URL, error)
}

type EndpointsFunc func(*http.Request) ([]*url.URL, error)

func (ef EndpointsFunc) NewEndpoints(original *http.Request) ([]*url.URL, error) {
	return ef(original)
}

// ShouldTerminateFunc is a strategy for determining if a fanout should terminate early given the results of
// a single HTTP transaction.
type ShouldTerminateFunc func(*http.Response, error) bool

// DefaultShouldTerminate is the default strategy for determining if an HTTP transaction should result
// in early termination of the fanout.  This function returns true if and only if fanoutResponse is non-nil
// and has a status code less than 400.
func DefaultShouldTerminate(fanoutResponse *http.Response, _ error) bool {
	return fanoutResponse != nil && fanoutResponse.StatusCode < 400
}

// FanoutRequestFunc is invoked to build a fanout request.  It can transfer information from the original request,
// set the body, update the context, etc.  This is the analog of go-kit's RequestFunc.
type FanoutRequestFunc func(ctx context.Context, original, fanout *http.Request, body []byte) context.Context

// OriginalBody creates a FanoutRequestFunc that makes the original request's body the body of each fanout request.
// If followRedirects is true, this function also sets fanout.GetBody so that the same body is read for redirects.
//
// This function also sets the ContentLength and Content-Type header appropriately.
func OriginalBody(followRedirects bool) FanoutRequestFunc {
	return func(ctx context.Context, original, fanout *http.Request, originalBody []byte) context.Context {
		fanout.ContentLength = int64(len(originalBody))
		fanout.Body = nil
		fanout.GetBody = nil
		fanout.Header.Del("Content-Type")

		if len(originalBody) > 0 {
			fanout.Header.Set("Content-Type", original.Header.Get("Content-Type"))
			body, getBody := xhttp.NewRewindBytes(originalBody)
			fanout.Body = body
			if followRedirects {
				fanout.GetBody = getBody
			}
		}

		return ctx
	}
}

// OriginalHeaders creates a FanoutRequestFunc that copies headers from the original request onto the fanout request
func OriginalHeaders(headers ...string) FanoutRequestFunc {
	canonicalizedHeaders := make([]string, len(headers))
	for i := 0; i < len(headers); i++ {
		canonicalizedHeaders[i] = textproto.CanonicalMIMEHeaderKey(headers[i])
	}

	return func(ctx context.Context, original, fanout *http.Request, _ []byte) context.Context {
		for _, key := range canonicalizedHeaders {
			if values := original.Header[key]; len(values) > 0 {
				fanout.Header[key] = append(fanout.Header[key], values...)
			}
		}

		return ctx
	}
}

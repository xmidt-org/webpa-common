package fanout

import (
	"context"
	"net/http"
	"net/textproto"

	"github.com/Comcast/webpa-common/xhttp"
)

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

// FanoutResponseFunc is a strategy applied to the termination fanout response.
type FanoutResponseFunc func(ctx context.Context, response http.ResponseWriter, result Result) context.Context

// FanoutHeaders copies zero or more headers from the fanout response into the top-level HTTP response.
func FanoutHeaders(headers ...string) FanoutResponseFunc {
	canonicalizedHeaders := make([]string, len(headers))
	for i := 0; i < len(headers); i++ {
		canonicalizedHeaders[i] = textproto.CanonicalMIMEHeaderKey(headers[i])
	}

	return func(ctx context.Context, response http.ResponseWriter, result Result) context.Context {
		if result.Response != nil {
			header := response.Header()
			for _, key := range canonicalizedHeaders {
				if values := result.Response.Header[key]; len(values) > 0 {
					header[key] = append(header[key], values...)
				}
			}
		}

		return ctx
	}
}

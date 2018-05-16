package fanout

import (
	"context"
	"net/http"
	"net/textproto"

	"github.com/Comcast/webpa-common/xhttp"
	"github.com/gorilla/mux"
)

// FanoutRequestFunc is invoked to build a fanout request.  It can transfer information from the original request,
// set the body, update the context, etc.  This is the analog of go-kit's RequestFunc.
type FanoutRequestFunc func(ctx context.Context, original, fanout *http.Request, body []byte) context.Context

// ForwardBody creates a FanoutRequestFunc that sends the original request's body to each fanout.
// If followRedirects is true, this function also sets fanout.GetBody so that the same body is read for redirects.
//
// This function also sets the ContentLength and Content-Type header appropriately.
func ForwardBody(followRedirects bool) FanoutRequestFunc {
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

// ForwardHeaders creates a FanoutRequestFunc that copies headers from the original request onto each fanout request
func ForwardHeaders(headers ...string) FanoutRequestFunc {
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

// ForwardVariableAsHeader returns a request function that copies the value of a gorilla/mux path variable
// from the original HTTP request into an HTTP header on each fanout request.
//
// The fanout request will always have the given header.  If no path variable is supplied (or no path variables
// are found), the fanout request will have the header associated with an empty string.
func ForwardVariableAsHeader(variable, header string) FanoutRequestFunc {
	return func(ctx context.Context, original, fanout *http.Request, _ []byte) context.Context {
		variables := mux.Vars(original)
		if len(variables) > 0 {
			fanout.Header.Add(header, variables[variable])
		} else {
			fanout.Header.Add(header, "")
		}

		return ctx
	}
}

// FanoutResponseFunc is a strategy applied to the termination fanout response.
type FanoutResponseFunc func(ctx context.Context, response http.ResponseWriter, result Result) context.Context

// ReturnHeaders copies zero or more headers from the fanout response into the top-level HTTP response.
func ReturnHeaders(headers ...string) FanoutResponseFunc {
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

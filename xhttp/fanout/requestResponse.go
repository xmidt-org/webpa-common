package fanout

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/textproto"

	"github.com/Comcast/webpa-common/xhttp"
	"github.com/gorilla/mux"
)

// BodyFunc is the strategy used to extract the body to send to each fanout endpoint
type BodyFunc func(context.Context, *http.Request) (context.Context, []byte, error)

// DefaultBodyFunc is the default strategy for extracting the body for each fanout request.
// This function simply extracts all the bytes from the original body.
func DefaultBodyFunc(ctx context.Context, original *http.Request) (context.Context, []byte, error) {
	if original.Body != nil {
		body, err := ioutil.ReadAll(original.Body)
		return ctx, body, err
	}

	return ctx, nil, nil
}

// RequestFunc is invoked to build a fanout request.  It can transfer information from the original request,
// set the body, update the context, etc.  This is the analog of go-kit's RequestFunc.
type RequestFunc func(ctx context.Context, original, fanout *http.Request, body []byte) (context.Context, error)

// ForwardBody creates a RequestFunc that sends the original request's body to each fanout.
// If followRedirects is true, this function also sets fanout.GetBody so that the same body is read for redirects.
//
// This function also sets the ContentLength and Content-Type header appropriately.
func ForwardBody(followRedirects bool) RequestFunc {
	return func(ctx context.Context, original, fanout *http.Request, originalBody []byte) (context.Context, error) {
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

		return ctx, nil
	}
}

// ForwardHeaders creates a RequestFunc that copies headers from the original request onto each fanout request
func ForwardHeaders(headers ...string) RequestFunc {
	canonicalizedHeaders := make([]string, len(headers))
	for i := 0; i < len(headers); i++ {
		canonicalizedHeaders[i] = textproto.CanonicalMIMEHeaderKey(headers[i])
	}

	return func(ctx context.Context, original, fanout *http.Request, _ []byte) (context.Context, error) {
		for _, key := range canonicalizedHeaders {
			if values := original.Header[key]; len(values) > 0 {
				fanout.Header[key] = append(fanout.Header[key], values...)
			}
		}

		return ctx, nil
	}
}

// UsePath sets a constant URI path for every fanout request.  Essentially, this replaces the original URL's
// Path with the configured value.
func UsePath(path string) RequestFunc {
	return func(ctx context.Context, original, fanout *http.Request, _ []byte) (context.Context, error) {
		fanout.URL.Path = path
		fanout.URL.RawPath = ""
		return ctx, nil
	}
}

// ForwardVariableAsHeader returns a request function that copies the value of a gorilla/mux path variable
// from the original HTTP request into an HTTP header on each fanout request.
//
// The fanout request will always have the given header.  If no path variable is supplied (or no path variables
// are found), the fanout request will have the header associated with an empty string.
func ForwardVariableAsHeader(variable, header string) RequestFunc {
	return func(ctx context.Context, original, fanout *http.Request, _ []byte) (context.Context, error) {
		variables := mux.Vars(original)
		if len(variables) > 0 {
			fanout.Header.Add(header, variables[variable])
		} else {
			fanout.Header.Add(header, "")
		}

		return ctx, nil
	}
}

// ResponseFunc is a strategy applied to the termination fanout response.
type ResponseFunc func(ctx context.Context, response http.ResponseWriter, result Result) context.Context

// ReturnHeaders copies zero or more headers from the fanout response into the top-level HTTP response.
func ReturnHeaders(headers ...string) ResponseFunc {
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

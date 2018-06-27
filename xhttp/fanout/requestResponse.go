package fanout

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/textproto"

	"github.com/gorilla/mux"
)

// DecoderFunc is the strategy used to decode the original request's message.  This decoder is passed a http.Header
// that can be modified.
//
// The DecoderFunc will be invoked once per HTTP request.  The returned byte slice and modified header will be passed
// to each fanout request.
type DecoderFunc func(ctx context.Context, original *http.Request, fanout http.Header) (context.Context, []byte, error)

// DefaultDecoderFunc is the default decoder strategy.  This returns the raw byte contents of the original request,
// and sets the Content-Type fanout header to the same value as the original request.
func DefaultDecoderFunc(ctx context.Context, original *http.Request, fanout http.Header) (context.Context, []byte, error) {
	body, err := ioutil.ReadAll(original.Body)
	if err != nil {
		return ctx, nil, err
	}

	contentType := original.Header.Get("Content-Type")
	if len(contentType) > 0 {
		fanout.Set("Content-Type", contentType)
	}

	return ctx, body, nil
}

// RequestFunc is invoked to add information to a fanout request.  This is the analog of go-kit's RequestFunc.
//
// Each RequestFunc is invoked once for each fanout request, after the information from the DecoderFunc
// has been applied to it.  In order to provide a strategy for decoding the original request, use a DecoderFunc.
type RequestFunc func(ctx context.Context, original, fanout *http.Request) (context.Context, error)

// ForwardHeaders creates a RequestFunc that copies headers from the original request onto each fanout request.
// By default, the Content-Type header is forwarded.
func ForwardHeaders(headers ...string) RequestFunc {
	canonicalizedHeaders := make([]string, len(headers))
	for i := 0; i < len(headers); i++ {
		canonicalizedHeaders[i] = textproto.CanonicalMIMEHeaderKey(headers[i])
	}

	return func(ctx context.Context, original, fanout *http.Request) (context.Context, error) {
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
	return func(ctx context.Context, _, fanout *http.Request) (context.Context, error) {
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
	return func(ctx context.Context, original, fanout *http.Request) (context.Context, error) {
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

func DefaultEncoderFunc(ctx context.Context, response http.ResponseWriter, result Result) context.Context {
	return ctx
}

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

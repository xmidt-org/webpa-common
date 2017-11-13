package fanouthttp

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/textproto"
	"net/url"
	"time"

	"github.com/Comcast/webpa-common/middleware/fanout"
	"github.com/Comcast/webpa-common/tracing"
	"github.com/go-kit/kit/endpoint"
	gokithttp "github.com/go-kit/kit/transport/http"
)

const (
	DefaultTimeout time.Duration = 30 * time.Second
)

// fanoutRequest is the internal type used to pass information to component requests.
// This type carries the original request so that downstream components can look at things
// like the header, the URL, etc.
type fanoutRequest struct {
	// original is the unmodified, original HTTP request passed to the fanout handler
	original *http.Request

	// relative is the original URL with absolute fields removed, i.e. Scheme, Host, and User.
	relative *url.URL

	// entity is the parsed HTTP entity returned by the configured DecodeRequestFunc
	entity interface{}
}

// CopyHeaders is a component client RequestFunc for transferring certain headers from the original
// request into each component request of a fanout.
//
// THe returned RequestFunc requires that the fanoutRequest is available in the context.
func CopyHeaders(headers ...string) gokithttp.RequestFunc {
	normalizedHeaders := make([]string, len(headers))
	for i, v := range headers {
		normalizedHeaders[i] = textproto.CanonicalMIMEHeaderKey(v)
	}

	headers = normalizedHeaders
	return func(ctx context.Context, r *http.Request) context.Context {
		if fr, ok := fanout.FromContext(ctx).(*fanoutRequest); ok {
			for _, name := range headers {
				if values, ok := fr.original.Header[name]; ok {
					r.Header[name] = values
				}
			}
		}

		return ctx
	}
}

// ExtraHeaders is a component client RequestFunc for setting extra headers for each component request.
func ExtraHeaders(extra http.Header) gokithttp.RequestFunc {
	normalizedExtra := make(http.Header, len(extra))
	for name, values := range extra {
		normalizedExtra[textproto.CanonicalMIMEHeaderKey(name)] = values
	}

	extra = normalizedExtra
	return func(ctx context.Context, r *http.Request) context.Context {
		for name, values := range extra {
			r.Header[name] = values
		}

		return ctx
	}
}

// decodeFanoutRequest is executed once per original request to turn an HTTP request into a fanoutRequest.
// The entityDecoder is used to perform one-time parsing on the original request to produce a custom entity object.
// If entityDecoder is nil, a decoder that simply returns the []byte contents of the HTTP entity is used instead.
func decodeFanoutRequest(dec gokithttp.DecodeRequestFunc) gokithttp.DecodeRequestFunc {
	if dec == nil {
		dec = func(_ context.Context, original *http.Request) (interface{}, error) {
			return ioutil.ReadAll(original.Body)
		}
	}

	return func(ctx context.Context, original *http.Request) (interface{}, error) {
		entity, err := dec(ctx, original)
		if err != nil {
			return nil, err
		}

		relative := *original.URL
		relative.Scheme = ""
		relative.Host = ""
		relative.User = nil

		return &fanoutRequest{
			original: original,
			relative: &relative,
			entity:   entity,
		}, nil
	}

}

// encodeComponentRequest creates the EncodeRequestFunc invoked for each component endpoint of a fanout.  Input to the
// return function is always a *fanoutRequest.
func encodeComponentRequest(enc gokithttp.EncodeRequestFunc) gokithttp.EncodeRequestFunc {
	return func(ctx context.Context, component *http.Request, v interface{}) error {
		fanoutRequest := v.(*fanoutRequest)

		component.Method = fanoutRequest.original.Method
		component.URL = component.URL.ResolveReference(fanoutRequest.relative)

		if enc != nil {
			return enc(ctx, component, fanoutRequest.entity)
		}

		return nil
	}
}

// NewComponents producces a mapped set of go-kit endpoints, one for each supplied URL.  Each endpoint is expected to accept
// a fanoutRequest.  However, the encoder function is only expected to decode the HTTP entity.  The fanoutRequest is never passed
// to the supplied encoder function.
//
// This factory function is the approximate equivalent of go-kit's transport/http.NewClient.  In effect, it creates a multi-client.
func NewComponents(urls []string, enc gokithttp.EncodeRequestFunc, dec gokithttp.DecodeResponseFunc, options ...gokithttp.ClientOption) (fanout.Components, error) {
	components := make(fanout.Components, len(urls))
	for _, raw := range urls {
		target, err := url.Parse(raw)
		if err != nil {
			return nil, err
		}

		if len(target.Scheme) == 0 {
			return nil, fmt.Errorf("Endpoint '%s' does not specify a scheme", raw)
		}

		if len(target.RawQuery) > 0 {
			return nil, fmt.Errorf("Endpoint '%s' specifies a query string", raw)
		}

		// the method and target don't really matter, since they'll be replaced on each
		// request with the appropriate information from the original HTTP request.
		components[raw] = gokithttp.NewClient(
			"GET",
			target,
			encodeComponentRequest(enc),
			dec,
			options...,
		).Endpoint()
	}

	return components, nil
}

// NewEndpoint returns an HTTP endpoint capable of fanning out to the supplied component endpoints.
// The result of this function can be decorated with middleware before using it to call NewFanoutHandler.
func NewEndpoint(components fanout.Components) endpoint.Endpoint {
	return fanout.New(
		tracing.NewSpanner(),
		components,
	)
}

// NewHandler creates an http.Handler (via go-kit's NewServer) that fans out requests
// to the configured endpoints.
//
// Note that the encode response function is used to encode the response from the first successful
// component request.
func NewHandler(endpoint endpoint.Endpoint, dec gokithttp.DecodeRequestFunc, enc gokithttp.EncodeResponseFunc, options ...gokithttp.ServerOption) http.Handler {
	return gokithttp.NewServer(
		endpoint,
		decodeFanoutRequest(dec),
		enc,
		options...,
	)
}

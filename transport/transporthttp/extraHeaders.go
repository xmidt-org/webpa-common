package transporthttp

import (
	"context"
	"net/http"
	"net/textproto"

	gokithttp "github.com/go-kit/kit/transport/http"
)

// ExtraHeaders is a RequestFunc for setting extra headers for each request.  This
// RequestFunc only makes sense for a client, as it modifies requests.
//
// The returned function has some performance advantages over go-kit's SetRequestHeader,
// particularly when more than one header is supplied.
//
// The header names are preprocessed using textproto.CanonicalMIMEHeaderKey just once.
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

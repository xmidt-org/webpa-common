package fanouthttp

import (
	"context"
	"net/http"
	"net/textproto"

	"github.com/Comcast/webpa-common/middleware/fanout"
	gokithttp "github.com/go-kit/kit/transport/http"
)

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

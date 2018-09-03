package xcontext

import (
	"context"
	"net/http"

	"github.com/Comcast/webpa-common/xhttp"
	gokithttp "github.com/go-kit/kit/transport/http"
)

// SetClient is a ContextFunc strategy that injects a supplied HTTP client into the HTTP context.
// Very useful when an outbound HTTP call needs to be made in response to a server request.
func SetClient(c xhttp.Client) gokithttp.RequestFunc {
	return func(ctx context.Context, _ *http.Request) context.Context {
		return xhttp.WithClient(ctx, c)
	}
}

package wrphttp

import (
	"context"
	"net/http"

	"github.com/Comcast/webpa-common/wrp"
	"github.com/Comcast/webpa-common/wrp/wrpendpoint"
	gokithttp "github.com/go-kit/kit/transport/http"
)

// ServerDecodeRequestBody creates a go-kit transport/http.DecodeRequestFunc function that parses the body of an HTTP
// request as a WRP message in the format used by the given pool.  The supplied pool should match the
// Content-Type of the request, or an error is returned.
//
// This decoder function is appropriate when the HTTP request body contains a full WRP message.  For situations
// where the HTTP body is only the payload, use the Headers decoder.
func ServerDecodeRequestBody(pool *wrp.DecoderPool) gokithttp.DecodeRequestFunc {
	return func(ctx context.Context, httpRequest *http.Request) (interface{}, error) {
		return wrpendpoint.DecodeRequest(ctx, httpRequest.Body, pool)
	}
}

// ServerDecodeHeaders uses headers to supply the WRP message fields.  The HTTP request body, if supplied, is assumed
// to be the payload of the WRP message and is read in unmodified.
func ServerDecodeRequestHeaders(ctx context.Context, httpRequest *http.Request) (interface{}, error) {
	message, err := NewMessageFromHeaders(httpRequest.Header, httpRequest.Body)
	if err != nil {
		return nil, err
	}

	return wrpendpoint.WrapAsRequest(ctx, message), nil
}

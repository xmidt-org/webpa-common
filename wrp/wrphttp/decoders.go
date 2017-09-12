package wrphttp

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/Comcast/webpa-common/wrp"
	"github.com/Comcast/webpa-common/wrp/wrpendpoint"
	gokithttp "github.com/go-kit/kit/transport/http"
)

// ClientDecodeResponseBody produces a go-kit transport/http.DecodeResponseFunc that turns an HTTP response
// into a WRP response.
func ClientDecodeResponseBody(pool *wrp.DecoderPool) gokithttp.DecodeResponseFunc {
	return func(ctx context.Context, httpResponse *http.Response) (interface{}, error) {
		body, err := ioutil.ReadAll(httpResponse.Body)
		if err != nil {
			return nil, err
		}

		if httpResponse.StatusCode == http.StatusOK {
			var (
				contentType         = httpResponse.Header.Get("Content-Type")
				responseFormat, err = wrp.FormatFromContentType(contentType)
			)

			if err != nil {
				return nil, err
			} else if responseFormat != pool.Format() {
				return nil, fmt.Errorf("Unexpected response Content-Type: %s", contentType)
			}

			return wrpendpoint.DecodeResponseBytes(body, pool)
		}

		return nil, fmt.Errorf("Error from WRP endpoint: %d", httpResponse.StatusCode)
	}
}

// ClientDecodeResponseHeaders is a go-kit transport/http.DecodeResponseFunc that turns an HTTP response
// formatted using headers for WRP fields into a WRP response.
func ClientDecodeResponseHeaders(ctx context.Context, httpResponse *http.Response) (interface{}, error) {
	body, err := ioutil.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, err
	}

	if httpResponse.StatusCode == http.StatusOK {
		message, err := NewMessageFromHeaders(httpResponse.Header, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}

		return wrpendpoint.WrapAsResponse(message), nil
	}

	return nil, fmt.Errorf("Error from WRP endpoint: %d", httpResponse.StatusCode)

}

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

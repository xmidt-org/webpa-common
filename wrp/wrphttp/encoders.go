package wrphttp

import (
	"bytes"
	"context"
	"net/http"

	"github.com/Comcast/webpa-common/wrp"
	"github.com/Comcast/webpa-common/wrp/wrpendpoint"
	gokithttp "github.com/go-kit/kit/transport/http"
)

// ServerEncodeResponseBody produces a go-kit transport/http.EncodeResponseFunc that transforms a wrphttp.Response into
// an HTTP response.  The returned function will transcode as necessary, based on the Format of the response.  If the WRP response
// has the same format as the output Format, the contents are passed through as is.
func ServerEncodeResponseBody(pool *wrp.EncoderPool) gokithttp.EncodeResponseFunc {
	return func(ctx context.Context, httpResponse http.ResponseWriter, value interface{}) error {
		var (
			wrpResponse = value.(wrpendpoint.Response)
			output      bytes.Buffer
		)

		if err := wrpResponse.Encode(&output, pool); err != nil {
			return err
		}

		httpResponse.Header().Set("Content-Type", pool.Format().ContentType())
		_, err := output.WriteTo(httpResponse)
		return err
	}
}

// ServerEncodeResponseHeaders encodes a WRP response's fields into the HTTP response's headers.  The payload
// is written as the HTTP response body.
func ServerEncodeResponseHeaders(ctx context.Context, httpResponse http.ResponseWriter, value interface{}) error {
	wrpResponse := value.(wrpendpoint.Response)
	AddMessageHeaders(httpResponse.Header(), wrpResponse.Message())
	return WriteMessagePayload(httpResponse.Header(), httpResponse, wrpResponse.Message())
}

package wrphttp

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"

	"github.com/Comcast/webpa-common/wrp"
	"github.com/Comcast/webpa-common/wrp/wrpendpoint"
	gokithttp "github.com/go-kit/kit/transport/http"
)

// ClientEncodeRequestBody produces a go-kit transport/http.EncodeRequestFunc for use when sending WRP requests
// to HTTP clients.  The returned decoder will set the appropriate headers and set the body to the encoded
// WRP message in the request.
func ClientEncodeRequestBody(pool *wrp.EncoderPool) gokithttp.EncodeRequestFunc {
	return func(ctx context.Context, httpRequest *http.Request, value interface{}) error {
		var (
			wrpRequest = value.(wrpendpoint.Request).WithContext(ctx)
			body       = new(bytes.Buffer)
		)

		if err := wrpRequest.Encode(body, pool); err != nil {
			return err
		}

		httpRequest.Header.Set(DestinationHeader, wrpRequest.Destination())
		httpRequest.Header.Set("Content-Type", pool.Format().ContentType())
		httpRequest.ContentLength = int64(body.Len())
		httpRequest.Body = ioutil.NopCloser(body)
		return nil
	}
}

// ClientEncodeRequestHeaders is a go-kit transport/http.EncodeRequestFunc for use when sending WRP requests
// to HTTP clients using an HTTP header representation of the message fields.
func ClientEncodeRequestHeaders(ctx context.Context, httpRequest *http.Request, value interface{}) error {
	var (
		wrpRequest = value.(wrpendpoint.Request).WithContext(ctx)
		body       = new(bytes.Buffer)
	)

	if err := WriteMessagePayload(httpRequest.Header, body, wrpRequest.Message()); err != nil {
		return err
	}

	AddMessageHeaders(httpRequest.Header, wrpRequest.Message())
	httpRequest.ContentLength = int64(body.Len())
	httpRequest.Body = ioutil.NopCloser(body)

	return nil
}

// ServerEncodeResponseBody produces a go-kit transport/http.EncodeResponseFunc that transforms a wrphttp.Response into
// an HTTP response.
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

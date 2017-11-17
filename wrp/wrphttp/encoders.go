package wrphttp

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"

	"github.com/Comcast/webpa-common/tracing"
	"github.com/Comcast/webpa-common/tracing/tracinghttp"
	"github.com/Comcast/webpa-common/wrp"
	"github.com/Comcast/webpa-common/wrp/wrpendpoint"
	gokithttp "github.com/go-kit/kit/transport/http"
)

// EncodeRequest returns a go-kit EncodeRequestFunc that encodes a decoded Entity as an HTTP request,
// often as the component of a fanout (though not required).  The given WRP format is used as the HTTP entity format.
func EncodeRequest(format wrp.Format) gokithttp.EncodeRequestFunc {
	return func(ctx context.Context, component *http.Request, v interface{}) error {
		entity := v.(*Entity)

		if format == entity.Format && len(entity.Contents) > 0 {
			// the entity is already formatted properly, so just write its contents out
			component.Body = ioutil.NopCloser(bytes.NewReader(entity.Contents))
			component.ContentLength = int64(len(entity.Contents))
		} else {
			var transcoded []byte
			if err := wrp.NewEncoderBytes(&transcoded, format).Encode(&entity.Message); err != nil {
				return err
			}

			component.Body = ioutil.NopCloser(bytes.NewReader(transcoded))
			component.ContentLength = int64(len(transcoded))
		}

		component.Header.Set("Content-Type", format.ContentType())
		component.Header.Set(DestinationHeader, entity.Message.Destination)
		return nil
	}
}

// ClientEncodeRequestBody produces a go-kit transport/http.EncodeRequestFunc for use when sending WRP requests
// to HTTP clients.  The returned decoder will set the appropriate headers and set the body to the encoded
// WRP message in the request.
func ClientEncodeRequestBody(pool *wrp.EncoderPool, custom http.Header) gokithttp.EncodeRequestFunc {
	return func(ctx context.Context, httpRequest *http.Request, value interface{}) error {
		var (
			wrpRequest = value.(wrpendpoint.Request)
			body       = new(bytes.Buffer)
		)

		if err := wrpRequest.Encode(body, pool); err != nil {
			return err
		}

		for name, values := range custom {
			for _, value := range values {
				httpRequest.Header.Add(name, value)
			}
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
func ClientEncodeRequestHeaders(custom http.Header) gokithttp.EncodeRequestFunc {
	return func(ctx context.Context, httpRequest *http.Request, value interface{}) error {
		var (
			wrpRequest = value.(wrpendpoint.Request)
			body       = new(bytes.Buffer)
		)

		if err := WriteMessagePayload(httpRequest.Header, body, wrpRequest.Message()); err != nil {
			return err
		}

		for name, values := range custom {
			for _, value := range values {
				httpRequest.Header.Add(name, value)
			}
		}

		AddMessageHeaders(httpRequest.Header, wrpRequest.Message())
		httpRequest.ContentLength = int64(body.Len())
		httpRequest.Body = ioutil.NopCloser(body)

		return nil
	}
}

// ServerEncodeResponseBody produces a go-kit transport/http.EncodeResponseFunc that transforms a wrphttp.Response into
// an HTTP response.
func ServerEncodeResponseBody(timeLayout string, pool *wrp.EncoderPool) gokithttp.EncodeResponseFunc {
	return func(ctx context.Context, httpResponse http.ResponseWriter, value interface{}) error {
		var (
			wrpResponse = value.(wrpendpoint.Response)
			output      bytes.Buffer
		)

		tracinghttp.HeadersForSpans(wrpResponse.Spans(), timeLayout, httpResponse.Header())

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
func ServerEncodeResponseHeaders(timeLayout string) gokithttp.EncodeResponseFunc {
	return func(ctx context.Context, httpResponse http.ResponseWriter, value interface{}) error {
		wrpResponse := value.(wrpendpoint.Response)
		tracinghttp.HeadersForSpans(wrpResponse.Spans(), timeLayout, httpResponse.Header())
		AddMessageHeaders(httpResponse.Header(), wrpResponse.Message())
		return WriteMessagePayload(httpResponse.Header(), httpResponse, wrpResponse.Message())
	}
}

// ServerErrorEncoder handles encoding the given error into an HTTP response, using the standard WebPA
// encoding for headers.
func ServerErrorEncoder(timeLayout string) gokithttp.ErrorEncoder {
	return func(ctx context.Context, err error, response http.ResponseWriter) {
		HeadersForError(err, timeLayout, response.Header())
		response.WriteHeader(StatusCodeForError(err))
	}
}

// HeadersForError provides the standard WRP/WebPA method for emitting HTTP response headers
// for an error object.
//
// (a) If err provides a Headers method that returns an http.Header, those headers are emitted
// (b) If err is a tracing.SpanError, the headers for the spans are written using tracinghttp.HeadersForSpans
//     and the causal error is passed to this function recursively.
// (c) Otherwise, no headers are written
func HeadersForError(err error, timeLayout string, h http.Header) {
	switch v := err.(type) {
	case gokithttp.Headerer:
		for name, values := range v.Headers() {
			for _, value := range values {
				h.Add(name, value)
			}
		}

	case tracing.SpanError:
		tracinghttp.HeadersForSpans(v.Spans(), timeLayout, h)
		HeadersForError(v.Err(), timeLayout, h)
	}
}

// StatusCodeForError implements the WRP/WebPA standard way of determining an HTTP response code
// from an error:
//
// (a) If err provides a StatusCode method, that value is returned
// (b) If err is a tracing.SpanError with more than 1 span, the individual component spans are examined to produce the code
// (c) If err is equal to context.DeadlineExceeded, http.StatusGatewayTimeout is returned
// (d) http.StatusInternalServerError is returned if no other case applies
func StatusCodeForError(err error) int {
	switch v := err.(type) {
	case gokithttp.StatusCoder:
		code := v.StatusCode()
		if code >= 500 {
			return http.StatusServiceUnavailable
		}

		return code

	case tracing.SpanError:
		cause := v.Err()
		if cause == context.DeadlineExceeded || cause == context.Canceled {
			return http.StatusGatewayTimeout
		}

		if spans := v.Spans(); len(spans) > 0 {
			for _, s := range spans {
				if e := s.Error(); e != nil {
					// recurse over the nested errors for each span
					// if any of these are in turn span errors, they'll be translated into
					// service unavailable for any timeouts
					code := StatusCodeForError(e)
					if code < 500 {
						return code
					}
				}
			}

			// if all spans indicate a 5XX error, then return service unavailable
			return http.StatusServiceUnavailable
		} else {
			// if the cause is not a context cancellation and there are no spans,
			// just recurse over the cause
			return StatusCodeForError(cause)
		}

	default:
		if err == context.DeadlineExceeded || err == context.Canceled {
			return http.StatusGatewayTimeout
		}
	}

	return http.StatusInternalServerError
}

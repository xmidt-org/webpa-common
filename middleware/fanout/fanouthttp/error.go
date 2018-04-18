package fanouthttp

import (
	"context"
	"net/http"

	"github.com/Comcast/webpa-common/tracing"
	"github.com/Comcast/webpa-common/tracing/tracinghttp"
	gokithttp "github.com/go-kit/kit/transport/http"
)

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

		return code

	case tracing.SpanError:
		cause := v.Err()
		if cause == context.DeadlineExceeded || cause == context.Canceled {
			return http.StatusGatewayTimeout
		}

		if spans := v.Spans(); len(spans) > 0 {
			largestCode := 0
			for _, s := range spans {
				if e := s.Error(); e != nil {
					// recurse over the nested errors for each span
					code := StatusCodeForError(e)
					if code > largestCode {
						largestCode = code
					}
				}
			}
			
			if largestCode > 0 {
				return largestCode
			}
			
			// if largestCode is still 0 then StatusServiceUnavailable
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

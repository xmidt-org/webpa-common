package fanout

import (
	"net/http"

	"github.com/xmidt-org/webpa-common/v2/tracing"
)

// Result is the result from a single fanout HTTP transaction
type Result struct {
	// StatusCode is the HTTP status code from the response, or an inferred status code
	// if the transaction returned an error.  This value will be populated even if Response is nil.
	StatusCode int

	// Request is the HTTP request sent to the fanout endpoint.  This will always be non-nil.
	Request *http.Request

	// Response is the HTTP response returned by the fanout HTTP transaction.  If set, Err will be nil.
	Response *http.Response

	// Err is the error returned by the fanout HTTP transaction.  If set, Response will be nil.
	Err error

	// ContentType is the MIME type of the Body
	ContentType string

	// Body is the HTTP response entity returned by the fanout HTTP transaction.  This can be nil or empty.
	Body []byte

	// Span represents the execution block that handled this fanout transaction
	Span tracing.Span
}

// ShouldTerminateFunc is a predicate for determining if a fanout should terminate early given the results of
// a single HTTP transaction.
type ShouldTerminateFunc func(Result) bool

// DefaultShouldTerminate is the default strategy for determining if an HTTP transaction should result
// in early termination of the fanout.  This function returns true if the status code is a non-error HTTP status.
func DefaultShouldTerminate(result Result) bool {
	return result.StatusCode < 400
}

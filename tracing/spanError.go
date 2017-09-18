package tracing

// SpanError represents an error that has one or more spans associated with it.  A SpanError
// augments an original error, accessible Err(), with zero or more spans.
//
// This error type also implements Mergeable from this package, allowing it to aggregate spans
// under a single causal error.
type SpanError interface {
	error
	Mergeable

	// Err returns the error object which is associated with the spans.  Error() returns
	// the value from this instance.
	Err() error
}

// NewSpanError "span-izes" an existing error object, returning the SpanError which
// annotates that error with one or more spans.
func NewSpanError(err error, spans ...Span) SpanError {
	return &spanError{
		err:   err,
		spans: spans,
	}
}

type spanError struct {
	err   error
	spans []Span
}

func (se *spanError) Error() string {
	return se.err.Error()
}

func (se *spanError) Spans() []Span {
	return se.spans
}

func (se *spanError) WithSpans(spans ...Span) interface{} {
	if len(spans) > 0 {
		return &spanError{
			err:   se.err,
			spans: spans,
		}
	}

	return se
}

func (se *spanError) Err() error {
	return se.err
}

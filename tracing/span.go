package tracing

import (
	"sync/atomic"
	"time"
)

// Spanned can be implemented by external messages to describe the spans
// involved in message processing.
type Spanned interface {
	Spans() []Span
}

// Span represents the result of some operation
type Span interface {
	// Name is the name of the operation
	Name() string

	// Start is the time at which the operation started
	Start() time.Time

	// Duration is how long the operation took.  Will be zero until Finish is called.
	Duration() time.Duration

	// Error is any error that occurred.  Will be nil until Finish is called, and then
	// it will be set to the error passed to Finish (which also can be nil).
	Error() error
}

type span struct {
	name     string
	start    time.Time
	duration time.Duration
	err      error

	state uint32
}

func (s *span) Name() string {
	return s.name
}

func (s *span) Start() time.Time {
	return s.start
}

func (s *span) Duration() time.Duration {
	return s.duration
}

func (s *span) Error() error {
	return s.err
}

func (s *span) finish(duration time.Duration, err error) bool {
	if atomic.CompareAndSwapUint32(&s.state, 0, 1) {
		s.duration = duration
		s.err = err
		return true
	}

	return false
}

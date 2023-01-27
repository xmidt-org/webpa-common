package tracing

import (
	// nolint: typecheck
	"sync/atomic"
	"time"
)

// Span represents the result of some arbitrary section of code.  Clients create Span objects
// via a Spanner.  A Span is immutable once it has been created via a Spanner closure.
type Span interface {
	// Name is the name of the operation
	Name() string

	// Start is the time at which the operation started
	Start() time.Time

	// Duration is how long the operation took.  This value is computed once, when the
	// closure from Spanner.Start is called.
	Duration() time.Duration

	// Error is any error that occurred.  This will be the error passed to the closure
	// returned from Spanner.Start.  This error can be nil.
	Error() error
}

// span is the internal Span implementation
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

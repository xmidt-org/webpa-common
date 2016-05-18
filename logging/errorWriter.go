package logging

import (
	"io"
)

// ErrorWriter adapts io.Writer onto ErrorLogger so that all output from Write() goes
// to Error(...).  This is useful for HTTP error logs.
type ErrorWriter struct {
	ErrorLogger
}

func (e *ErrorWriter) Write(data []byte) (int, error) {
	e.Error(string(data))
	return len(data), nil
}

var _ io.Writer = (*ErrorWriter)(nil)

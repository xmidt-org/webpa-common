package logging

import (
	"io"

	"github.com/go-kit/kit/log"
)

// testLogger is implemented by testing.T and testing.B
type testLogger interface {
	Log(...interface{})
}

// testWriter implements io.Writer and delegates to a testLogger
type testWriter struct {
	testLogger
}

func (t testWriter) Write(data []byte) (int, error) {
	t.testLogger.Log(string(data))
	return len(data), nil
}

// NewTestWriter returns an io.Writer which delegates to a testing log.
// The returned io.Writer does not need to be synchronized.
func NewTestWriter(t testLogger) io.Writer {
	return testWriter{t}
}

// NewTestLogger produces a go-kit Logger which delegates to the supplied testing log.
func NewTestLogger(o *Options, t testLogger) log.Logger {
	if o == nil {
		// we want to see all log output in tests by default
		o = &Options{Level: "DEBUG"}
	}

	return NewFilter(
		log.With(
			o.loggerFactory()(NewTestWriter(t)),
			TimestampKey(), log.DefaultTimestampUTC,
		),
		o,
	)
}

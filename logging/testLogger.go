package logging

import (
	"io"
	"sync"

	"github.com/go-kit/kit/log"
)

// testSink is implemented by testing.T and testing.B
type testSink interface {
	Log(...interface{})
}

// testWriter implements io.Writer and delegates to a testSink
type testWriter struct {
	mux sync.Mutex
	testSink
}

func (t *testWriter) Write(data []byte) (int, error) {
	t.mux.Lock()
	defer t.mux.Unlock()
	cpy := make([]byte, len(data))
	copy(cpy, data)
	t.testSink.Log(string(cpy))
	return len(cpy), nil
}

// NewTestWriter returns an io.Writer which delegates to a testing log.
// The returned io.Writer does not need to be synchronized.
func NewTestWriter(t testSink) io.Writer {
	return &testWriter{testSink: t}
}

// NewTestLogger produces a go-kit Logger which delegates to the supplied testing log.
func NewTestLogger(o *Options, t testSink) log.Logger {
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

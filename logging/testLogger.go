package logging

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"github.com/go-kit/kit/log"
)

// testSink is implemented by testing.T and testing.B
type testSink interface {
	Log(...interface{})
}

// testWriter implements io.Writer
type testWriter struct {
	mux sync.Mutex
}

func (t *testWriter) Write(data []byte) (int, error) {
	t.mux.Lock()
	defer t.mux.Unlock()
	fmt.Fprint(os.Stdout, string(data))
	return len(data), nil
}

// NewTestWriter returns an io.Writer which writes to stdout
// only when testing in verbose mode.
// The returned io.Writer does not need to be synchronized.
// Note: Although originally intended to delegate data to testSink,
// intermittent data races have forced us to stick to writing directly
// to stdout and do the verbose check outselves.
func NewTestWriter(_ testSink) io.Writer {
	if !testing.Verbose() {
		return ioutil.Discard
	}
	return new(testWriter)
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

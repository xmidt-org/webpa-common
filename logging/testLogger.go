package logging

import (
	"os"
	"testing"

	"github.com/go-kit/kit/log"
)

// testSink is implemented by testing.T and testing.B
type testSink interface {
	Log(...interface{})
}

// NewTestLogger produces a go-kit Logger which logs to stdout only if
// the verbose testing mode.
// Note: Although originally intended to delegate data to testSink,
// intermittent data races have forced us to stick to writing directly
// to stdout and do the verbose check outselves.
func NewTestLogger(o *Options, _ testSink) log.Logger {
	if !testing.Verbose() {
		return log.NewNopLogger()
	}

	if o == nil {
		// we want to see all log output in tests by default
		o = &Options{Level: "DEBUG"}
	}

	return NewFilter(
		log.With(
			o.loggerFactory()(log.NewSyncWriter(os.Stdout)),
			TimestampKey(), log.DefaultTimestampUTC,
		),
		o,
	)
}

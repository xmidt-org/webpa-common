package logging

import (
	"fmt"

	"github.com/go-kit/kit/log"
)

// CaptureLogger is a go-kit Logger which dispatches log key/value pairs to a channel
// for test assertions and verifications.  Primarily useful for test code.
//
// The Log method of this type will panic if the number of key/value pairs is not odd, which
// is appropriate for test code.
type CaptureLogger interface {
	log.Logger

	// Output returns the channel on which each log event is recorded as a map of key/value pairs
	Output() <-chan map[interface{}]interface{}
}

type captureLogger struct {
	output chan map[interface{}]interface{}
}

func (cl *captureLogger) Output() <-chan map[interface{}]interface{} {
	return cl.output
}

func (cl *captureLogger) Log(kv ...interface{}) error {
	if len(kv)%2 != 0 {
		panic(fmt.Errorf("Invalid key/value count: %d", len(kv)))
	}

	m := make(map[interface{}]interface{}, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}

	cl.output <- m
	return nil
}

func NewCaptureLogger() CaptureLogger {
	return &captureLogger{
		output: make(chan map[interface{}]interface{}, 10),
	}
}

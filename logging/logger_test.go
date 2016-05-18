package logging

import (
	"bytes"
	"testing"
)

type testStringer struct {
	message string
}

func (t testStringer) String() string {
	return t.message
}

func TestLoggerWriterUsingParameters(t *testing.T) {
	var usingParameters = []struct {
		parameters      []interface{}
		expectedMessage string
	}{
		{
			[]interface{}{},
			"",
		},
		{
			[]interface{}{"this is a format string: %d", 12},
			"this is a format string: 12",
		},
		{
			[]interface{}{"a %s complicated %d format string", "foobar", -23},
			"a foobar complicated -23 format string",
		},
		{
			[]interface{}{testStringer{""}},
			"",
		},
		{
			[]interface{}{testStringer{"this is a format string: %d"}, 12},
			"this is a format string: 12",
		},
		{
			[]interface{}{testStringer{"a %s complicated %d format string"}, "foobar", -23},
			"a foobar complicated -23 format string",
		},
		{
			[]interface{}{47},
			"47",
		},
		{
			[]interface{}{-1234, "rawk! I shouldn't be!"},
			"-1234%!(EXTRA string=rawk! I shouldn't be!)",
		},
	}

	var output bytes.Buffer
	loggerWriter := LoggerWriter{&output}
	verify := func(expectedLogEntry string, logFunction func(...interface{}), parameters []interface{}) {
		output.Reset()
		logFunction(parameters...)
		if expectedLogEntry != output.String() {
			t.Errorf(`Expected "%s", but got "%s"`, expectedLogEntry, output.String())
		}
	}

	for _, record := range usingParameters {
		verify(traceLevel+record.expectedMessage+"\n", loggerWriter.Trace, record.parameters)
		verify(debugLevel+record.expectedMessage+"\n", loggerWriter.Debug, record.parameters)
		verify(infoLevel+record.expectedMessage+"\n", loggerWriter.Info, record.parameters)
		verify(warnLevel+record.expectedMessage+"\n", loggerWriter.Warn, record.parameters)
		verify(errorLevel+record.expectedMessage+"\n", loggerWriter.Error, record.parameters)
	}
}

func TestLoggerWriterUsingFormat(t *testing.T) {
	var formats = []struct {
		format          string
		parameters      []interface{}
		expectedMessage string
	}{
		{
			"",
			nil,
			"",
		},
		{
			"%s",
			[]interface{}{"foobar"},
			"foobar",
		},
		{
			"%s: %d",
			[]interface{}{"foobar", 12},
			"foobar: 12",
		},
	}

	var output bytes.Buffer
	loggerWriter := LoggerWriter{&output}
	verify := func(expectedLogEntry string, formatFunction func(string, ...interface{}), format string, parameters []interface{}) {
		output.Reset()
		formatFunction(format, parameters...)
		if expectedLogEntry != output.String() {
			t.Errorf(`Expected "%s", but got "%s"`, expectedLogEntry, output.String())
		}
	}

	for _, record := range formats {
		verify(infoLevel+record.expectedMessage+"\n", loggerWriter.Printf, record.format, record.parameters)
	}
}

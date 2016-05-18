package logging

import (
	"testing"
)

type mockErrorLogger struct {
	t        *testing.T
	expected string
}

func (mock mockErrorLogger) Error(parameters ...interface{}) {
	if len(parameters) != 1 {
		mock.t.Errorf("Invalid number of parameters passed to logger.Errorf: %d", len(parameters))
	}

	actual := parameters[0].(string)
	if mock.expected != actual {
		mock.t.Errorf("Expected %s, but got %s", mock.expected, actual)
	}
}

func (mock mockErrorLogger) Errorf(format string, parameters ...interface{}) {
	mock.t.Errorf("Unexpected call to logger.Errorf")
}

func TestErrorWriter(t *testing.T) {
	var testData = []struct {
		errorMessage string
	}{
		{""},
		{"here is a lovely error"},
	}

	for _, record := range testData {
		mock := mockErrorLogger{t, record.errorMessage}
		errorWriter := &ErrorWriter{mock}
		errorWriter.Write([]byte(record.errorMessage))
	}
}

package logging

import (
	"bytes"
	"fmt"
	"testing"
)

func TestErrorWriter(t *testing.T) {
	testErrorMessage := []byte("this is an error message!")
	var output bytes.Buffer
	errorLogger := DefaultLogger{&output}
	errorWriter := ErrorWriter{errorLogger}

	count, err := errorWriter.Write(testErrorMessage)
	if err != nil {
		t.Errorf("Failed to write test error message: %v", err)
	}

	if count != len(testErrorMessage) {
		t.Errorf("Invalid count returned from Write(): %d", count)
	}

	actualErrorMessage := output.String()
	expectedErrorMessage := fmt.Sprintf("[ERROR] %s\n", testErrorMessage)
	if actualErrorMessage != expectedErrorMessage {
		t.Errorf(`"Expected error "%s", but got "%s"`, expectedErrorMessage, actualErrorMessage)
	}
}

type Message struct {
	string
}

func (m Message) String() string {
	return m.string
}

func TestDefaultLogger(t *testing.T) {
	var testData = []struct {
		parameters []interface{}
		message    string
	}{
		{[]interface{}{}, ""},
		{[]interface{}{"this is a message"}, "this is a message"},
		{[]interface{}{"this is a message that uses a parameter: %d", 123}, "this is a message that uses a parameter: 123"},
		{[]interface{}{Message{"here is a stringer"}}, "here is a stringer"},
		{[]interface{}{Message{"here is a stringer with a parameter: %s"}, "foobar"}, "here is a stringer with a parameter: foobar"},
	}

	var buffer bytes.Buffer
	defaultLogger := DefaultLogger{&buffer}
	for _, record := range testData {
		buffer.Reset()
		defaultLogger.Debug(record.parameters...)
		actualMessage := buffer.String()
		expectedMessage := fmt.Sprintf("[%-5.5s] %s\n", "DEBUG", record.message)
		if expectedMessage != actualMessage {
			t.Errorf(`"Expected debug message "%s", but got "%s"`, expectedMessage, actualMessage)
		}

		buffer.Reset()
		defaultLogger.Info(record.parameters...)
		actualMessage = buffer.String()
		expectedMessage = fmt.Sprintf("[%-5.5s] %s\n", "INFO", record.message)
		if expectedMessage != actualMessage {
			t.Errorf(`"Expected info message "%s", but got "%s"`, expectedMessage, actualMessage)
		}

		buffer.Reset()
		defaultLogger.Warn(record.parameters...)
		actualMessage = buffer.String()
		expectedMessage = fmt.Sprintf("[%-5.5s] %s\n", "WARN", record.message)
		if expectedMessage != actualMessage {
			t.Errorf(`"Expected warn message "%s", but got "%s"`, expectedMessage, actualMessage)
		}

		buffer.Reset()
		defaultLogger.Error(record.parameters...)
		actualMessage = buffer.String()
		expectedMessage = fmt.Sprintf("[%-5.5s] %s\n", "ERROR", record.message)
		if expectedMessage != actualMessage {
			t.Errorf(`"Expected error message "%s", but got "%s"`, expectedMessage, actualMessage)
		}

		buffer.Reset()
		defaultLogger.Fatal(record.parameters...)
		actualMessage = buffer.String()
		expectedMessage = fmt.Sprintf("[%-5.5s] %s\n", "FATAL", record.message)
		if expectedMessage != actualMessage {
			t.Errorf(`"Expected fatal message "%s", but got "%s"`, expectedMessage, actualMessage)
		}
	}
}

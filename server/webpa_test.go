package server

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	unexpectedListenAndServe    string = "Unexpected call to ListenAndServe"
	unexpectedListenAndServeTLS string = "Unexpected call to ListenAndServeTLS"
)

// testServerExecutor provides a test implementation of serverExecutor
type testServerExecutor struct {
	t                       *testing.T
	expectListenAndServe    func(t *testing.T) error
	expectListenAndServeTLS func(t *testing.T, certificateFile, keyFile string) error
}

func (e *testServerExecutor) ListenAndServe() error {
	if e.expectListenAndServe == nil {
		e.t.Errorf(unexpectedListenAndServe)
		return errors.New(unexpectedListenAndServe)
	}

	return e.expectListenAndServe(e.t)
}

func (e *testServerExecutor) ListenAndServeTLS(certificateFile, keyFile string) error {
	if e.expectListenAndServeTLS == nil {
		e.t.Errorf(unexpectedListenAndServeTLS)
		return errors.New(unexpectedListenAndServeTLS)
	}

	return e.expectListenAndServeTLS(e.t, certificateFile, keyFile)
}

func TestWebPAMarshalJSON(t *testing.T) {
	assertions := assert.New(t)

	var testData = []struct {
		webPA    webPA
		expected string
	}{
		{
			webPA:    webPA{},
			expected: `{"name": "", "address": "", "cert": "", "key": ""}`,
		},
		{
			webPA: webPA{
				name:    "foobar",
				address: ":8080",
			},
			expected: `{"name": "foobar", "address": ":8080", "cert": "", "key": ""}`,
		},
		{
			webPA: webPA{
				name:            "moomar",
				address:         ":9191",
				certificateFile: "/etc/config/somefile.cert",
				keyFile:         "/etc/config/somefile.pem",
			},
			expected: `{"name": "moomar", "address": ":9191", "cert": "/etc/config/somefile.cert", "key": "/etc/config/somefile.pem"}`,
		},
	}

	for _, record := range testData {
		data, err := record.webPA.MarshalJSON()
		if err != nil {
			t.Fatalf("Failed to marshal JSON: %v", err)
		}

		if !assertions.JSONEq(record.expected, string(data)) {
			t.Errorf("Expected JSON %s, but got %s", data, record.expected)
		}

		stringValue := record.webPA.String()
		if string(data) != stringValue {
			t.Errorf("Expected string value %s, but got %s", data, stringValue)
		}
	}
}

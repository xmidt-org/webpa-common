package server

import (
	"fmt"
	"github.com/Comcast/webpa-common/concurrent"
	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
	"os"
	"sync"
	"testing"
	"time"
)

func TestWebPAMarshalJSON(t *testing.T) {
	assertions := assert.New(t)

	var testData = []struct {
		webPA    *WebPA
		expected string
	}{
		{
			&WebPA{},
			`{"name": "", "address": "", "cert": "", "key": ""}`,
		},
		{
			&WebPA{
				name:    "foobar",
				address: ":8080",
			},
			`{"name": "foobar", "address": ":8080", "cert": "", "key": ""}`,
		},
		{
			&WebPA{
				name:            "moomar",
				address:         ":9191",
				certificateFile: expectedCertificateFile,
				keyFile:         expectedKeyFile,
			},
			fmt.Sprintf(
				`{"name": "moomar", "address": ":9191", "cert": "%s", "key": "%s"}`,
				expectedCertificateFile,
				expectedKeyFile,
			),
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

func TestWebPARun(t *testing.T) {
	var testData = []struct {
		webPA WebPA
	}{
		{
			WebPA{
				name:    "foobar",
				address: ":8080",
				serverExecutor: &testServerExecutor{
					t:                    t,
					expectListenAndServe: func(*testing.T) error { return nil },
				},
				logger: &logging.LoggerWriter{os.Stdout},
			},
		},
		{
			WebPA{
				name:            "foobar",
				address:         ":8080",
				certificateFile: expectedCertificateFile,
				keyFile:         expectedKeyFile,
				serverExecutor: &testServerExecutor{
					t: t,
					expectListenAndServeTLS: func(t *testing.T, certificateFile, keyFile string) error {
						if expectedCertificateFile != certificateFile {
							t.Errorf("Expected certificate file %s, but got %s", expectedCertificateFile, certificateFile)
						}

						if expectedKeyFile != keyFile {
							t.Errorf("Expected key file %s, but got %s", expectedKeyFile, keyFile)
						}

						return nil
					},
				},
				logger: &logging.LoggerWriter{os.Stdout},
			},
		},
	}

	for _, record := range testData {
		waitGroup := &sync.WaitGroup{}
		shutdown := make(chan struct{})
		defer close(shutdown)
		err := record.webPA.Run(waitGroup, shutdown)
		if err != nil {
			t.Errorf("Failed to run webPA instance: %v", err)
		}

		if !concurrent.WaitTimeout(waitGroup, time.Second*5) {
			t.Errorf("WaitGroup.Done() was not called within the timeout")
		}
	}
}

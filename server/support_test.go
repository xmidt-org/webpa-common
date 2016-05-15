package server

import (
	"errors"
	"github.com/Comcast/webpa-common/logging"
	"testing"
)

const (
	unexpectedListenAndServe    string = "Unexpected call to ListenAndServe"
	unexpectedListenAndServeTLS string = "Unexpected call to ListenAndServeTLS"
	unexpectedNewLogger         string = "Unexpected call to NewLogger"
	expectedCertificateFile     string = "/etc/myapp/cert"
	expectedKeyFile             string = "/etc/myapp/cert"
)

// testServerExecutor provides a test implementation of serverExecutor
type testServerExecutor struct {
	t                       *testing.T
	expectListenAndServe    func(t *testing.T) error
	expectListenAndServeTLS func(t *testing.T, certificateFile, keyFile string) error
}

func (executor *testServerExecutor) ListenAndServe() error {
	if executor.expectListenAndServe == nil {
		executor.t.Errorf(unexpectedListenAndServe)
		return errors.New(unexpectedListenAndServe)
	}

	return executor.expectListenAndServe(executor.t)
}

func (executor *testServerExecutor) ListenAndServeTLS(certificateFile, keyFile string) error {
	if executor.expectListenAndServeTLS == nil {
		executor.t.Errorf(unexpectedListenAndServeTLS)
		return errors.New(unexpectedListenAndServeTLS)
	}

	return executor.expectListenAndServeTLS(executor.t, certificateFile, keyFile)
}

// testLoggerFactory provides a mocked logging.LoggerFactory for testing
type testLoggerFactory struct {
	t               *testing.T
	expectNewLogger func(*testing.T, string) (logging.Logger, error)
}

func (factory *testLoggerFactory) NewLogger(name string) (logging.Logger, error) {
	if factory.expectNewLogger == nil {
		factory.t.Errorf(unexpectedNewLogger)
		return nil, errors.New(unexpectedNewLogger)
	}

	return factory.expectNewLogger(factory.t, name)
}

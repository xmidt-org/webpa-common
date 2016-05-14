package server

import (
	"errors"
	"testing"
)

const (
	unexpectedListenAndServe    string = "Unexpected call to ListenAndServe"
	unexpectedListenAndServeTLS string = "Unexpected call to ListenAndServeTLS"
	expectedCertificateFile     string = "/etc/myapp/cert"
	expectedKeyFile             string = "/etc/myapp/cert"
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

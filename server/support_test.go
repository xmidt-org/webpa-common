package server

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/Comcast/webpa-common/logging"
	"net"
	"testing"
	"time"
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
	t             *testing.T
	buffer        bytes.Buffer
	expectedNames map[string]bool
}

func (factory *testLoggerFactory) NewLogger(name string) (logging.Logger, error) {
	if !factory.expectedNames[name] {
		message := fmt.Sprintf("Unexpected logger name %s", name)
		factory.t.Errorf(message)
		return nil, errors.New(message)
	}

	return &logging.LoggerWriter{&factory.buffer}, nil
}

func newTestLoggerFactory(t *testing.T, expectedNames ...string) *testLoggerFactory {
	loggerFactory := &testLoggerFactory{t: t, expectedNames: make(map[string]bool, len(expectedNames))}
	for _, name := range expectedNames {
		loggerFactory.expectedNames[name] = true
	}

	return loggerFactory
}

// mockConn implements both net.Conn and net.Addr, for testing
type mockConn struct {
	t *testing.T
}

func (m mockConn) Network() string {
	return "network"
}

func (m mockConn) String() string {
	return "127.0.0.1"
}

func (m mockConn) Read(b []byte) (n int, err error) {
	m.t.Fatal("Read should not have been called")
	return 0, nil
}

func (m mockConn) Write(b []byte) (n int, err error) {
	m.t.Fatal("Write should not have been called")
	return 0, nil
}

func (m mockConn) Close() error {
	m.t.Fatal("Close should not have been called")
	return nil
}

func (m mockConn) LocalAddr() net.Addr {
	return m
}

func (m mockConn) RemoteAddr() net.Addr {
	m.t.Fatal("RemoteAddr should not have been called")
	return nil
}

func (m mockConn) SetDeadline(t time.Time) error {
	m.t.Fatal("SetDeadline should not have been called")
	return nil
}

func (m mockConn) SetReadDeadline(t time.Time) error {
	m.t.Fatal("SetReadDeadline should not have been called")
	return nil
}

func (m mockConn) SetWriteDeadline(t time.Time) error {
	m.t.Fatal("SetWriteDeadline should not have been called")
	return nil
}

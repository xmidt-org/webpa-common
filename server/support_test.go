package server

import (
	"errors"
	"net"
	"testing"
	"time"
)

const (
	unexpectedListenAndServe    string = "Unexpected call to ListenAndServe"
	unexpectedListenAndServeTLS string = "Unexpected call to ListenAndServeTLS"
	unexpectedNewLogger         string = "Unexpected call to NewLogger"
	expectedCertificateFile     string = "/etc/myapp/cert"
	expectedKeyFile             string = "/etc/myapp/key"
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

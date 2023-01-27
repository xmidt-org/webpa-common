package xhttp

import (
	"net"

	"github.com/stretchr/testify/mock"
)

type mockReader struct {
	mock.Mock
}

func (m *mockReader) Read(b []byte) (int, error) {
	// nolint: typecheck
	arguments := m.Called(b)
	return arguments.Int(0), arguments.Error(1)
}

type mockHTTPServer struct {
	mock.Mock
}

func (m *mockHTTPServer) ListenAndServe() error {
	// nolint: typecheck
	return m.Called().Error(0)
}

func (m *mockHTTPServer) ListenAndServeTLS(certificateFile, keyFile string) error {
	// nolint: typecheck
	return m.Called(certificateFile, keyFile).Error(0)
}

func (m *mockHTTPServer) Serve(l net.Listener) error {
	// nolint: typecheck
	return m.Called(l).Error(0)
}

func (m *mockHTTPServer) ServeTLS(l net.Listener, certificateFile, keyFile string) error {
	// nolint: typecheck
	return m.Called(l, certificateFile, keyFile).Error(0)
}

func (m *mockHTTPServer) SetKeepAlivesEnabled(v bool) {
	// nolint: typecheck
	m.Called(v)
}

type mockListener struct {
	mock.Mock
}

func (m *mockListener) Accept() (net.Conn, error) {
	// nolint: typecheck
	arguments := m.Called()
	first, _ := arguments.Get(0).(net.Conn)
	return first, arguments.Error(1)
}

func (m *mockListener) Close() error {
	// nolint: typecheck
	return m.Called().Error(0)
}

func (m *mockListener) Addr() net.Addr {
	// nolint: typecheck
	return m.Called().Get(0).(net.Addr)
}

type mockTempError struct{}

func (m mockTempError) Temporary() bool { return true }

func (m mockTempError) Error() string { return "mock temp error" }

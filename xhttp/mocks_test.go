package xhttp

import (
	"net"

	"github.com/stretchr/testify/mock"
)

type mockReader struct {
	mock.Mock
}

func (m *mockReader) Read(b []byte) (int, error) {
	arguments := m.Called(b)
	return arguments.Int(0), arguments.Error(1)
}

type mockHTTPServer struct {
	mock.Mock
}

func (m *mockHTTPServer) ListenAndServe() error {
	return m.Called().Error(0)
}

func (m *mockHTTPServer) ListenAndServeTLS(certificateFile, keyFile string) error {
	return m.Called(certificateFile, keyFile).Error(0)
}

func (m *mockHTTPServer) Serve(l net.Listener) error {
	return m.Called(l).Error(0)
}

func (m *mockHTTPServer) ServeTLS(l net.Listener, certificateFile, keyFile string) error {
	return m.Called(l, certificateFile, keyFile).Error(0)
}

func (m *mockHTTPServer) SetKeepAlivesEnabled(v bool) {
	m.Called(v)
}

type mockListener struct {
	mock.Mock
}

func (m *mockListener) Accept() (net.Conn, error) {
	arguments := m.Called()
	first, _ := arguments.Get(0).(net.Conn)
	return first, arguments.Error(1)
}

func (m *mockListener) Close() error {
	return m.Called().Error(0)
}

func (m *mockListener) Addr() net.Addr {
	return m.Called().Get(0).(net.Addr)
}

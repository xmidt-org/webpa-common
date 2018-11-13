package server

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/stretchr/testify/mock"
)

type mockHandler struct {
	mock.Mock
}

func (m *mockHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	m.Called(response, request)
}

type mockExecutor struct {
	mock.Mock
}

func (m *mockExecutor) Serve(l net.Listener) error {
	return m.Called(l).Error(0)
}

func (m *mockExecutor) ServeTLS(l net.Listener, certificateFile, keyFile string) error {
	return m.Called(l, certificateFile, keyFile).Error(0)
}

func (m *mockExecutor) ListenAndServe() error {
	return m.Called().Error(0)
}

func (m *mockExecutor) ListenAndServeTLS(certificateFile, keyFile string) error {
	return m.Called(certificateFile, keyFile).Error(0)
}

func (m *mockExecutor) Shutdown(ctx context.Context) error {
	return m.Called(ctx).Error(0)
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

type mockConn struct {
	mock.Mock
}

func (m *mockConn) Read(b []byte) (int, error) {
	arguments := m.Called(b)
	return arguments.Int(0), arguments.Error(1)
}

func (m *mockConn) Write(b []byte) (int, error) {
	arguments := m.Called(b)
	return arguments.Int(0), arguments.Error(1)
}

func (m *mockConn) Close() error {
	return m.Called().Error(0)
}

func (m *mockConn) LocalAddr() net.Addr {
	return m.Called().Get(0).(net.Addr)
}

func (m *mockConn) RemoteAddr() net.Addr {
	return m.Called().Get(0).(net.Addr)
}

func (m *mockConn) SetDeadline(t time.Time) error {
	return m.Called(t).Error(0)
}

func (m *mockConn) SetReadDeadline(t time.Time) error {
	return m.Called(t).Error(0)
}

func (m *mockConn) SetWriteDeadline(t time.Time) error {
	return m.Called(t).Error(0)
}

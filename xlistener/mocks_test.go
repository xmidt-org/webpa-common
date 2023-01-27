package xlistener

import (
	"net"
	"time"

	"github.com/stretchr/testify/mock"
)

type mockConn struct {
	mock.Mock
}

func (m *mockConn) Read(b []byte) (int, error) {
	// nolint: typecheck
	arguments := m.Called(b)
	return arguments.Int(0), arguments.Error(1)
}

func (m *mockConn) Write(b []byte) (int, error) {
	// nolint: typecheck
	arguments := m.Called(b)
	return arguments.Int(0), arguments.Error(1)
}

func (m *mockConn) Close() error {
	// nolint: typecheck
	return m.Called().Error(0)
}

func (m *mockConn) LocalAddr() net.Addr {
	// nolint: typecheck
	return m.Called().Get(0).(net.Addr)
}

func (m *mockConn) RemoteAddr() net.Addr {
	// nolint: typecheck
	return m.Called().Get(0).(net.Addr)
}

func (m *mockConn) SetDeadline(t time.Time) error {
	// nolint: typecheck
	return m.Called(t).Error(0)
}

func (m *mockConn) SetReadDeadline(t time.Time) error {
	// nolint: typecheck
	return m.Called(t).Error(0)
}

func (m *mockConn) SetWriteDeadline(t time.Time) error {
	// nolint: typecheck
	return m.Called(t).Error(0)
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

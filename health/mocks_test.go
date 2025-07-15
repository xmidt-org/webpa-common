// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package health

import (
	"bufio"
	"net"
	"net/http"

	"github.com/stretchr/testify/mock"
)

type mockHandler struct {
	mock.Mock
}

func (m *mockHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	m.Called(response, request)
}

// mockResponseWriter is a type that only mocks http.ResponseWriter
type mockResponseWriter struct {
	mock.Mock
}

func (m *mockResponseWriter) Header() http.Header {
	return m.Called().Get(0).(http.Header)
}

func (m *mockResponseWriter) Write(data []byte) (int, error) {
	arguments := m.Called(data)
	return arguments.Int(0), arguments.Error(1)
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	m.Called(statusCode)
}

// mockResponseWriterFull is a type that not only mocks http.ResponseWriter but also
// mocks http.CloseNotifier, http.Hijacker, http.Pusher, and http.Flusher.
type mockResponseWriterFull struct {
	mock.Mock
}

func (m *mockResponseWriterFull) Header() http.Header {
	return m.Called().Get(0).(http.Header)
}

func (m *mockResponseWriterFull) Write(data []byte) (int, error) {
	arguments := m.Called(data)
	return arguments.Int(0), arguments.Error(1)
}

func (m *mockResponseWriterFull) WriteHeader(statusCode int) {
	m.Called(statusCode)
}

func (m *mockResponseWriterFull) CloseNotify() <-chan bool {
	first, _ := m.Called().Get(0).(<-chan bool)
	return first
}

func (m *mockResponseWriterFull) Flush() {
	m.Called()
}

func (m *mockResponseWriterFull) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	var (
		arguments = m.Called()
		first, _  = arguments.Get(0).(net.Conn)
		second, _ = arguments.Get(1).(*bufio.ReadWriter)
	)

	return first, second, arguments.Error(2)
}

func (m *mockResponseWriterFull) Push(target string, opts *http.PushOptions) error {
	return m.Called(target, opts).Error(0)
}

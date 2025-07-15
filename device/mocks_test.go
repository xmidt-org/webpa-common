// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package device

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockReader is a mocked io.Reader
type mockReader struct {
	mock.Mock
}

func (m *mockReader) Read(b []byte) (int, error) {
	// nolint: typecheck
	arguments := m.Called(b)
	return arguments.Int(0), arguments.Error(1)
}

// mockConnectionReader is a mocked Reader, from this package.  It represents
// the read side of a websocket.// nolint: typecheck
type mockConnectionReader struct {
	mock.Mock
}

func (m *mockConnectionReader) ReadMessage() (int, []byte, error) {
	// nolint: typecheck
	arguments := m.Called()
	return arguments.Int(0), arguments.Get(1).([]byte), arguments.Error(2)
}

func (m *mockConnectionReader) SetReadDeadline(d time.Time) error {
	// nolint: typecheck
	return m.Called(d).Error(0)
}

func (m *mockConnectionReader) SetPongHandler(h func(string) error) {
	// nolint: typecheck
	m.Called(h)
}

func (m *mockConnectionReader) Close() error {
	// nolint: typecheck
	return m.Called().Error(0)
}

// mockConnectionWriter is a mocked Writer, from this package.  It represents
// the write side of a websocket.
type mockConnectionWriter struct {
	mock.Mock
}

func (m *mockConnectionWriter) WriteMessage(messageType int, data []byte) error {
	// nolint: typecheck
	return m.Called(messageType, data).Error(0)
}

func (m *mockConnectionWriter) WritePreparedMessage(pm *websocket.PreparedMessage) error {
	// nolint: typecheck
	return m.Called(pm).Error(0)
}

func (m *mockConnectionWriter) SetWriteDeadline(d time.Time) error {
	// nolint: typecheck
	return m.Called(d).Error(0)
}

func (m *mockConnectionWriter) Close() error {
	// nolint: typecheck
	return m.Called().Error(0)
}

type mockDialer struct {
	mock.Mock
}

func (m *mockDialer) DialDevice(deviceName, url string, extra http.Header) (*websocket.Conn, *http.Response, error) {
	var (
		// nolint: typecheck
		arguments = m.Called(deviceName, url, extra)
		first, _  = arguments.Get(0).(*websocket.Conn)
		second, _ = arguments.Get(1).(*http.Response)
	)

	return first, second, arguments.Error(2)
}

type mockWebsocketDialer struct {
	mock.Mock
}

func (m *mockWebsocketDialer) Dial(url string, requestHeader http.Header) (*websocket.Conn, *http.Response, error) {
	var (
		// nolint: typecheck
		arguments = m.Called(url, requestHeader)
		first, _  = arguments.Get(0).(*websocket.Conn)
		second, _ = arguments.Get(1).(*http.Response)
	)

	return first, second, arguments.Error(2)
}

// deviceSet is a convenient map type for capturing visited devices
// and asserting expectations.
type deviceSet map[*device]bool

func (s deviceSet) len() int {
	return len(s)
}

func (s deviceSet) add(d Interface) {
	s[d.(*device)] = true
}

func (s *deviceSet) reset() {
	*s = make(deviceSet)
}

// managerCapture returns a high-level visitor for Manager testing
func (s deviceSet) managerCapture() func(Interface) bool {
	return func(d Interface) bool {
		s.add(d)
		return true
	}
}

// drain copies whatever is available on the given channel into this device set
func (s deviceSet) drain(source <-chan Interface) {
	for d := range source {
		s.add(d)
	}
}

type mockRouter struct {
	mock.Mock
}

func (m *mockRouter) Route(request *Request) (*Response, error) {
	// nolint: typecheck
	arguments := m.Called(request)
	first, _ := arguments.Get(0).(*Response)
	return first, arguments.Error(1)
}

func TestMockConnector(t *testing.T) {
	var (
		assert = assert.New(t)

		c = new(MockConnector)

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "/", nil)
		// nolint: typecheck
		header               = http.Header{"X-Something": {"foo"}}
		expectedDevice       = new(MockDevice)
		expectedConnectError = errors.New("expected connect error")

		id1 = ID("test1")
		id2 = ID("test2")

		predicateCalled = false
		predicate       = func(candidate ID) (CloseReason, bool) {
			predicateCalled = true
			return CloseReason{}, false
		}
	)

	// nolint: typecheck
	c.On("Connect", response, request, header).Return(expectedDevice, expectedConnectError).Once()
	// nolint: typecheck
	c.On("Disconnect", id1, CloseReason{}).Return(true).Once()
	// nolint: typecheck
	c.On("Disconnect", id2, CloseReason{}).Return(false).Once()
	// nolint: typecheck
	c.On("DisconnectIf", mock.MatchedBy(func(func(ID) (CloseReason, bool)) bool { return true })).Return(5).
		Run(func(arguments mock.Arguments) {
			arguments.Get(0).(func(ID) (CloseReason, bool))(id1)
		}).Once()
	// nolint: typecheck
	c.On("DisconnectAll", CloseReason{}).Return(12).Once()

	actualDevice, actualConnectError := c.Connect(response, request, header)
	assert.Equal(expectedDevice, actualDevice)
	assert.Equal(expectedConnectError, actualConnectError)

	assert.True(c.Disconnect(id1, CloseReason{}))
	assert.False(c.Disconnect(id2, CloseReason{}))

	assert.Equal(5, c.DisconnectIf(predicate))
	assert.True(predicateCalled)

	assert.Equal(12, c.DisconnectAll(CloseReason{}))

	// nolint: typecheck
	c.AssertExpectations(t)
}

type mockFilter struct {
	mock.Mock
}

func (m *mockFilter) AllowConnection(d Interface) (bool, MatchResult) {
	// nolint: typecheck
	args := m.Called(d)
	result, _ := args.Get(1).(MatchResult)
	return args.Bool(0), result
}

// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package device

import (
	"errors"
	"net/http"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDialerDefault(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		result = NewDialer(DialerOptions{})
	)

	require.NotNil(result)
	d, ok := result.(*dialer)
	require.True(ok)

	assert.Equal(DeviceNameHeader, d.deviceHeader)
	assert.Equal(defaultWebsocketDialer, d.wd)
}

func testDialerDialDevice(t *testing.T, deviceName, expectedURL, deviceHeader string, extra, expectedHeader http.Header) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedConn     = new(websocket.Conn)
		expectedResponse = new(http.Response)

		websocketDialer = new(mockWebsocketDialer)
		dialer          = NewDialer(DialerOptions{DeviceHeader: deviceHeader, WSDialer: websocketDialer})
	)

	require.NotNil(dialer)

	// nolint: typecheck
	websocketDialer.On("Dial", expectedURL, expectedHeader).
		Return(expectedConn, expectedResponse, nil).
		Once()

	// nolint:bodyclose
	actualConn, actualResponse, actualErr := dialer.DialDevice(deviceName, expectedURL, extra)
	assert.True(expectedConn == actualConn)
	assert.True(expectedResponse == actualResponse)
	assert.Nil(actualErr)

	// nolint: typecheck
	websocketDialer.AssertExpectations(t)
}

func TestDialer(t *testing.T) {
	t.Run("Default", testDialerDefault)

	const (
		deviceName  = "mac:112233445566/service"
		expectedURL = "http://somewhere.foobar.com/api/blah/blah"
	)

	t.Run("DialDevice", func(t *testing.T) {
		// nolint: typecheck
		testDialerDialDevice(t, deviceName, expectedURL, "", nil, http.Header{DeviceNameHeader: {deviceName}})
		// nolint: typecheck
		testDialerDialDevice(t, deviceName, expectedURL, "X-Something", http.Header{"Content-Type": {"text/plain"}}, http.Header{"Content-Type": {"text/plain"}, "X-Something": {deviceName}})
	})
}

func testMustDialDeviceSuccess(t *testing.T, deviceName, url string, extra http.Header) {
	var (
		assert           = assert.New(t)
		expectedConn     = new(websocket.Conn)
		expectedResponse = new(http.Response)

		dialer = new(mockDialer)
	)

	// nolint: typecheck
	dialer.On("DialDevice", deviceName, url, extra).
		Return(expectedConn, expectedResponse, nil).
		Once()

	assert.NotPanics(func() {
		// nolint:bodyclose
		actualConn, actualResponse := MustDialDevice(dialer, deviceName, url, extra)
		assert.True(expectedConn == actualConn)
		assert.True(expectedResponse == actualResponse)
	})

	// nolint: typecheck
	dialer.AssertExpectations(t)
}

func testMustDialDeviceFailure(t *testing.T, deviceName, url string, extra http.Header) {
	var (
		assert        = assert.New(t)
		expectedError = errors.New("expected panic")

		dialer = new(mockDialer)
	)

	// nolint: typecheck
	dialer.On("DialDevice", deviceName, url, extra).
		Return(nil, nil, expectedError).
		Once()

	assert.Panics(func() {

		// nolint:bodyclose
		MustDialDevice(dialer, deviceName, url, extra)
	})

	// nolint: typecheck
	dialer.AssertExpectations(t)
}

func TestMustDialDevice(t *testing.T) {
	const (
		deviceName  = "mac:112233445566/service"
		expectedURL = "http://somewhere.foobar.com/api/blah/blah"
	)

	t.Run("Success", func(t *testing.T) {
		testMustDialDeviceSuccess(t, deviceName, expectedURL, nil)

		// nolint: typecheck
		testMustDialDeviceSuccess(t, deviceName, expectedURL, http.Header{"Content-Type": {"text/plain"}, "X-Something": {"value1", "value2"}})
	})

	t.Run("Failure", func(t *testing.T) {
		testMustDialDeviceFailure(t, deviceName, expectedURL, nil)

		// nolint: typecheck
		testMustDialDeviceFailure(t, deviceName, expectedURL, http.Header{"Content-Type": {"text/plain"}, "X-Something": {"value1", "value2"}})
	})
}

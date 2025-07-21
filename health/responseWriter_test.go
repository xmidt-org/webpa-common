// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package health

import (
	"bufio"
	"bytes"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testResponseWriterWrapping(t *testing.T) {
	var (
		assert    = assert.New(t)
		delegate  = new(mockResponseWriter)
		composite = Wrap(delegate)
	)

	delegate.On("WriteHeader", 200).Once()

	assert.Equal(0, composite.StatusCode())
	composite.WriteHeader(200)
	assert.Equal(200, composite.StatusCode())

	delegate.AssertExpectations(t)
}

func testResponseWriterCloseNotifier(t *testing.T) {
	assert := assert.New(t)

	{
		var (
			delegate                     = new(mockResponseWriter)
			composite http.CloseNotifier = Wrap(delegate)
		)

		assert.Panics(func() { composite.CloseNotify() })
		delegate.AssertExpectations(t)
	}

	{
		var (
			delegate                     = new(mockResponseWriterFull)
			composite http.CloseNotifier = Wrap(delegate)

			closeChannel                = make(chan bool, 1)
			expectedChannel <-chan bool = closeChannel
		)

		delegate.On("CloseNotify").Return(expectedChannel).Once()
		assert.Equal(expectedChannel, composite.CloseNotify())
		delegate.AssertExpectations(t)
	}
}

func testResponseWriterHijacker(t *testing.T) {
	var (
		assert                = assert.New(t)
		expectedConn net.Conn = &net.IPConn{}

		buffer             bytes.Buffer
		expectedReadWriter = bufio.NewReadWriter(
			bufio.NewReader(&buffer),
			bufio.NewWriter(&buffer),
		)
	)

	{
		var (
			delegate                = new(mockResponseWriter)
			composite http.Hijacker = Wrap(delegate)
		)

		conn, rw, err := composite.Hijack()
		assert.Nil(conn)
		assert.Nil(rw)
		assert.Error(err)

		delegate.AssertExpectations(t)
	}

	{
		var (
			delegate                = new(mockResponseWriterFull)
			composite http.Hijacker = Wrap(delegate)
		)

		delegate.On("Hijack").Return(expectedConn, expectedReadWriter, error(nil)).Once()
		conn, rw, err := composite.Hijack()
		assert.Equal(expectedConn, conn)
		assert.Equal(expectedReadWriter, rw)
		assert.NoError(err)

		delegate.AssertExpectations(t)
	}
}

func testResponseWriterFlusher(t *testing.T) {
	{
		var (
			delegate               = new(mockResponseWriter)
			composite http.Flusher = Wrap(delegate)
		)

		composite.Flush()
		delegate.AssertExpectations(t)
	}

	{
		var (
			delegate               = new(mockResponseWriterFull)
			composite http.Flusher = Wrap(delegate)
		)

		delegate.On("Flush").Once()
		composite.Flush()
		delegate.AssertExpectations(t)
	}
}

func testResponseWriterPusher(t *testing.T) {
	var (
		assert          = assert.New(t)
		expectedTarget  = "expectedTarget"
		expectedOptions = &http.PushOptions{Method: "GET"}
	)

	{
		var (
			delegate              = new(mockResponseWriter)
			composite http.Pusher = Wrap(delegate)
		)

		assert.Error(composite.Push(expectedTarget, expectedOptions))
		delegate.AssertExpectations(t)
	}

	{
		var (
			delegate              = new(mockResponseWriterFull)
			composite http.Pusher = Wrap(delegate)
		)

		delegate.On("Push", expectedTarget, expectedOptions).Return(error(nil)).Once()
		assert.NoError(composite.Push(expectedTarget, expectedOptions))
		delegate.AssertExpectations(t)
	}
}

func TestResponseWriter(t *testing.T) {
	t.Run("Wrapping", testResponseWriterWrapping)
	t.Run("CloseNotifier", testResponseWriterCloseNotifier)
	t.Run("Hijacker", testResponseWriterHijacker)
	t.Run("Flusher", testResponseWriterFlusher)
	t.Run("Pusher", testResponseWriterPusher)
}

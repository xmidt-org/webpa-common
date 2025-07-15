// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package xlistener

import (
	"crypto/tls"
	"errors"
	"net"
	"testing"

	"github.com/go-kit/kit/metrics/generic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/sallust"
)

func testNewDefault(t *testing.T) {
	defer func() { netListen = net.Listen }()

	var (
		assert       = assert.New(t)
		require      = require.New(t)
		expectedNext = new(mockListener)
	)

	// nolint: typecheck
	expectedNext.On("Addr").Return(new(net.IPAddr)).Twice()

	netListen = func(network, address string) (net.Listener, error) {
		assert.Equal("tcp", network)
		assert.Equal(":http", address)
		return expectedNext, nil
	}

	l, err := New(Options{})
	require.NoError(err)
	require.NotNil(l)

	assert.Equal(expectedNext, l.(*listener).Listener)
	assert.NotNil(l.(*listener).logger)
	assert.Nil(l.(*listener).semaphore)
	assert.NotNil(l.(*listener).rejected)
	assert.NotNil(l.(*listener).active)

	// nolint: typecheck
	expectedNext.AssertExpectations(t)
}

func testNewCustom(t *testing.T) {
	defer func() { netListen = net.Listen }()

	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedRejected = generic.NewCounter("test")
		expectedActive   = generic.NewGauge("test")
		expectedNext     = new(mockListener)
	)

	// nolint: typecheck
	expectedNext.On("Addr").Return(new(net.IPAddr)).Twice()

	netListen = func(network, address string) (net.Listener, error) {
		assert.Equal("tcp4", network)
		assert.Equal(":8080", address)
		return expectedNext, nil
	}

	l, err := New(Options{
		Logger:         sallust.Default(),
		Rejected:       expectedRejected,
		Active:         expectedActive,
		Network:        "tcp4",
		Address:        ":8080",
		MaxConnections: 10,
	})

	require.NoError(err)
	require.NotNil(l)

	assert.Equal(expectedNext, l.(*listener).Listener)
	assert.NotNil(l.(*listener).logger)
	assert.NotNil(l.(*listener).semaphore)

	require.NotNil(l.(*listener).rejected)
	l.(*listener).rejected.Inc()
	assert.Equal(1.0, expectedRejected.Value())

	require.NotNil(l.(*listener).active)
	l.(*listener).active.Add(10.0)
	assert.Equal(10.0, expectedActive.Value())

	// nolint: typecheck
	expectedNext.AssertExpectations(t)
}

func testNewTLSCustom(t *testing.T) {
	defer func() { netListen = net.Listen }()

	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedRejected = generic.NewCounter("test")
		expectedActive   = generic.NewGauge("test")
		expectedNext     = new(mockListener)
	)

	// nolint: typecheck
	expectedNext.On("Addr").Return(new(net.IPAddr)).Twice()

	tlsListen = func(network, address string, config *tls.Config) (net.Listener, error) {
		assert.Equal("tcp4", network)
		assert.Equal(":8080", address)
		assert.Equal(true, config.InsecureSkipVerify)
		return expectedNext, nil
	}

	l, err := New(Options{
		Logger:         sallust.Default(),
		Rejected:       expectedRejected,
		Active:         expectedActive,
		Network:        "tcp4",
		Address:        ":8080",
		MaxConnections: 10,
		Config: &tls.Config{
			InsecureSkipVerify: true,
		},
	})

	require.NoError(err)
	require.NotNil(l)

	assert.Equal(expectedNext, l.(*listener).Listener)
	assert.NotNil(l.(*listener).logger)
	assert.NotNil(l.(*listener).semaphore)

	require.NotNil(l.(*listener).rejected)
	l.(*listener).rejected.Inc()
	assert.Equal(1.0, expectedRejected.Value())

	require.NotNil(l.(*listener).active)
	l.(*listener).active.Add(10.0)
	assert.Equal(10.0, expectedActive.Value())

	// nolint: typecheck
	expectedNext.AssertExpectations(t)
}

func testNewListenError(t *testing.T) {
	defer func() { netListen = net.Listen }()

	var (
		assert        = assert.New(t)
		expectedError = errors.New("expected")
	)

	netListen = func(network, address string) (net.Listener, error) {
		assert.Equal("tcp", network)
		assert.Equal(":http", address)
		return nil, expectedError
	}

	l, actualError := New(Options{})
	assert.Nil(l)
	assert.Equal(expectedError, actualError)
}

func TestNew(t *testing.T) {
	t.Run("Default", testNewDefault)
	t.Run("Custom", testNewCustom)
	t.Run("tlsCustom", testNewTLSCustom)
	t.Run("ListenError", testNewListenError)
}

func testListenerAcceptError(t *testing.T, maxConnections int) {
	defer func() { netListen = net.Listen }()

	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedRejected = generic.NewCounter("test")
		expectedActive   = generic.NewGauge("test")
		expectedError    = errors.New("expected")
		expectedNext     = new(mockListener)
	)

	// nolint: typecheck
	expectedNext.On("Addr").Return(new(net.IPAddr)).Twice()
	// nolint: typecheck
	expectedNext.On("Accept").Return(nil, expectedError).Once()

	l, err := New(Options{
		Logger:         sallust.Default(),
		MaxConnections: maxConnections,
		Rejected:       expectedRejected,
		Active:         expectedActive,
		Next:           expectedNext,
	})

	require.NoError(err)
	require.NotNil(l)

	c, actualError := l.Accept()
	assert.Nil(c)
	assert.Equal(expectedError, actualError)
	assert.Equal(0.0, expectedRejected.Value())
	assert.Equal(0.0, expectedActive.Value())

	// nolint: typecheck
	expectedNext.AssertExpectations(t)
}

func testListenerAcceptUnlimitedConnections(t *testing.T) {
	defer func() { netListen = net.Listen }()

	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedRejected = generic.NewCounter("test")
		expectedActive   = generic.NewGauge("test")
		expectedNext     = new(mockListener)

		expectedConn1          = new(mockConn)
		expectedConn2          = new(mockConn)
		expectedConnCloseError = errors.New("expected")
	)

	// nolint: typecheck
	expectedNext.On("Addr").Return(new(net.IPAddr)).Twice()
	// nolint: typecheck
	expectedConn1.On("RemoteAddr").Return(new(net.IPAddr)).Once()
	// nolint: typecheck
	expectedConn2.On("RemoteAddr").Return(new(net.IPAddr)).Once()

	// nolint: typecheck
	expectedNext.On("Accept").Return(expectedConn1, error(nil)).Once()
	// nolint: typecheck
	expectedNext.On("Accept").Return(expectedConn2, error(nil)).Once()

	// nolint: typecheck
	expectedConn1.On("Close").Return(error(nil)).Once()
	// nolint: typecheck
	expectedConn1.On("Close").Return(expectedConnCloseError).Once()
	// nolint: typecheck
	expectedConn2.On("Close").Return(error(nil)).Once()
	// nolint: typecheck
	expectedConn2.On("Close").Return(expectedConnCloseError).Once()

	l, err := New(Options{
		Logger:   sallust.Default(),
		Rejected: expectedRejected,
		Active:   expectedActive,
		Next:     expectedNext,
	})

	require.NoError(err)
	require.NotNil(l)

	assert.Zero(expectedRejected.Value())
	assert.Zero(expectedActive.Value())

	actualConn1, actualError := l.Accept()
	assert.NoError(actualError)
	require.NotNil(actualConn1)
	assert.Zero(expectedRejected.Value())
	assert.Equal(1.0, expectedActive.Value())

	actualConn2, actualError := l.Accept()
	assert.NoError(actualError)
	require.NotNil(actualConn2)
	assert.Zero(expectedRejected.Value())
	assert.Equal(2.0, expectedActive.Value())

	assert.NoError(actualConn1.Close())
	assert.Zero(expectedRejected.Value())
	assert.Equal(1.0, expectedActive.Value())

	assert.Equal(expectedConnCloseError, actualConn1.Close())
	assert.Zero(expectedRejected.Value())
	assert.Equal(1.0, expectedActive.Value())

	assert.NoError(actualConn2.Close())
	assert.Zero(expectedRejected.Value())
	assert.Zero(expectedActive.Value())

	assert.Equal(expectedConnCloseError, actualConn2.Close())
	assert.Zero(expectedRejected.Value())
	assert.Zero(expectedActive.Value())

	// nolint: typecheck
	expectedNext.AssertExpectations(t)
	// nolint: typecheck
	expectedConn1.AssertExpectations(t)
	// nolint: typecheck
	expectedConn2.AssertExpectations(t)
}

func testListenerAcceptMaxConnections(t *testing.T) {
	defer func() { netListen = net.Listen }()

	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedRejected = generic.NewCounter("test")
		expectedActive   = generic.NewGauge("test")
		expectedNext     = new(mockListener)

		expectedConn1          = new(mockConn)
		rejectedConn           = new(mockConn)
		expectedConn2          = new(mockConn)
		expectedConnCloseError = errors.New("expected close error")
		expectedAcceptError    = errors.New("expected accept error")
	)

	// nolint: typecheck
	expectedNext.On("Addr").Return(new(net.IPAddr)).Twice()
	// nolint: typecheck
	expectedConn1.On("RemoteAddr").Return(new(net.IPAddr)).Once()
	// nolint: typecheck
	rejectedConn.On("RemoteAddr").Return(new(net.IPAddr)).Once()
	// nolint: typecheck
	expectedConn2.On("RemoteAddr").Return(new(net.IPAddr)).Once()

	// nolint: typecheck
	expectedNext.On("Accept").Return(expectedConn1, error(nil)).Once()
	// nolint: typecheck
	expectedNext.On("Accept").Return(rejectedConn, error(nil)).Once()
	// nolint: typecheck
	expectedNext.On("Accept").Return(nil, expectedAcceptError).Once()
	// nolint: typecheck
	expectedNext.On("Accept").Return(expectedConn2, error(nil)).Once()

	// nolint: typecheck
	expectedConn1.On("Close").Return(error(nil)).Once()
	// nolint: typecheck
	expectedConn1.On("Close").Return(expectedConnCloseError).Once()
	// nolint: typecheck
	rejectedConn.On("Close").Return(error(nil)).Once() // this should be closed as part of rejecting the connection
	// nolint: typecheck
	expectedConn2.On("Close").Return(error(nil)).Once()
	// nolint: typecheck
	expectedConn2.On("Close").Return(expectedConnCloseError).Once()

	l, err := New(Options{
		Logger:         sallust.Default(),
		MaxConnections: 1,
		Rejected:       expectedRejected,
		Active:         expectedActive,
		Next:           expectedNext,
	})

	require.NoError(err)
	require.NotNil(l)

	assert.Zero(expectedRejected.Value())
	assert.Zero(expectedActive.Value())

	actualConn1, actualError := l.Accept()
	assert.NoError(actualError)
	require.NotNil(actualConn1)
	assert.Zero(expectedRejected.Value())
	assert.Equal(1.0, expectedActive.Value())

	actualRejectedConn, actualError := l.Accept()
	assert.Equal(expectedAcceptError, actualError)
	assert.Nil(actualRejectedConn)
	assert.Equal(1.0, expectedRejected.Value())
	assert.Equal(1.0, expectedActive.Value())

	assert.NoError(actualConn1.Close())
	assert.Equal(1.0, expectedRejected.Value())
	assert.Zero(expectedActive.Value())

	assert.Equal(expectedConnCloseError, actualConn1.Close())
	assert.Equal(1.0, expectedRejected.Value())
	assert.Zero(expectedActive.Value())

	// now, a new connection should be possible
	actualConn2, actualError := l.Accept()
	assert.NoError(actualError)
	require.NotNil(actualConn2)
	assert.Equal(1.0, expectedRejected.Value())
	assert.Equal(1.0, expectedActive.Value())

	assert.NoError(actualConn2.Close())
	assert.Equal(1.0, expectedRejected.Value())
	assert.Zero(expectedActive.Value())

	assert.Equal(expectedConnCloseError, actualConn2.Close())
	assert.Equal(1.0, expectedRejected.Value())
	assert.Zero(expectedActive.Value())

	// nolint: typecheck
	expectedNext.AssertExpectations(t)
	// nolint: typecheck
	expectedConn1.AssertExpectations(t)
	// nolint: typecheck
	rejectedConn.AssertExpectations(t)
	// nolint: typecheck
	expectedConn2.AssertExpectations(t)
}

func TestListener(t *testing.T) {
	t.Run("Accept", func(t *testing.T) {
		t.Run("Error", func(t *testing.T) {
			t.Run("UnlimitedConnections", func(t *testing.T) { testListenerAcceptError(t, 0) })
			t.Run("MaxConnections", func(t *testing.T) { testListenerAcceptError(t, 1) })
		})

		t.Run("Success", func(t *testing.T) {
			t.Run("UnlimitedConnections", testListenerAcceptUnlimitedConnections)
			t.Run("MaxConnections", testListenerAcceptMaxConnections)
		})
	})
}

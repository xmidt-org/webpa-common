package xlistener

import (
	"errors"
	"net"
	"testing"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/xmetrics"
	"github.com/go-kit/kit/metrics/generic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testNewDefault(t *testing.T) {
	defer func() { netListen = net.Listen }()

	var (
		assert       = assert.New(t)
		require      = require.New(t)
		expectedNext = new(mockListener)
		listenAddr   = new(mockAddr)
	)

	listenAddr.On("Network").Return("tcp").Once()
	listenAddr.On("String").Return(":http").Once()
	expectedNext.On("Addr").Return(listenAddr).Twice()

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

	expectedNext.AssertExpectations(t)
	listenAddr.AssertExpectations(t)
}

func testNewCustom(t *testing.T) {
	defer func() { netListen = net.Listen }()

	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedRejected = generic.NewCounter("test")
		expectedActive   = generic.NewGauge("test")
		expectedNext     = new(mockListener)
		listenAddr       = new(mockAddr)
	)

	listenAddr.On("Network").Return("tcp4").Once()
	listenAddr.On("String").Return(":8080").Once()
	expectedNext.On("Addr").Return(listenAddr).Twice()

	netListen = func(network, address string) (net.Listener, error) {
		assert.Equal("tcp4", network)
		assert.Equal(":8080", address)
		return expectedNext, nil
	}

	l, err := New(Options{
		Logger:         logging.NewTestLogger(nil, t),
		Rejected:       xmetrics.NewIncrementer(expectedRejected),
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

	expectedNext.AssertExpectations(t)
	listenAddr.AssertExpectations(t)
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
		listenAddr       = new(mockAddr)
	)

	listenAddr.On("Network").Return("tcp").Once()
	listenAddr.On("String").Return(":http").Once()
	expectedNext.On("Addr").Return(listenAddr).Twice()
	expectedNext.On("Accept").Return(nil, expectedError).Once()

	l, err := New(Options{
		Logger:         logging.NewTestLogger(nil, t),
		MaxConnections: maxConnections,
		Rejected:       xmetrics.NewIncrementer(expectedRejected),
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

	listenAddr.AssertExpectations(t)
	expectedNext.AssertExpectations(t)
}

func TestListener(t *testing.T) {
	t.Run("AcceptError", func(t *testing.T) {
		t.Run("UnlimitedConnections", func(t *testing.T) { testListenerAcceptError(t, 0) })
		t.Run("MaxConnections", func(t *testing.T) { testListenerAcceptError(t, 1) })
	})
}

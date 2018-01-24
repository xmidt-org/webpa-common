package server

import (
	"errors"
	"testing"

	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/metrics/generic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testInstrumentListenerClose(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		logger   = logging.NewTestLogger(nil, t)
		gauge    = generic.NewGauge("test")
		delegate = new(mockListener)
		listener = InstrumentListener(logger, gauge, delegate)
	)

	require.NotNil(listener)

	delegate.On("Close").Return(error(nil)).Once()

	assert.Nil(listener.Close())
	assert.Zero(gauge.Value())

	delegate.AssertExpectations(t)
}

func testInstrumentListenerCloseError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedError = errors.New("expected error from Close")
		logger        = logging.NewTestLogger(nil, t)
		gauge         = generic.NewGauge("test")
		delegate      = new(mockListener)
		listener      = InstrumentListener(logger, gauge, delegate)
	)

	require.NotNil(listener)

	delegate.On("Close").Return(expectedError).Once()

	assert.Equal(expectedError, listener.Close())
	assert.Zero(gauge.Value())

	delegate.AssertExpectations(t)
}

func testInstrumentListenerAccept(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		logger       = logging.NewTestLogger(nil, t)
		gauge        = generic.NewGauge("test")
		delegate     = new(mockListener)
		expectedConn = new(mockConn)
		listener     = InstrumentListener(logger, gauge, delegate)
	)

	require.NotNil(listener)

	delegate.On("Accept").Return(expectedConn, error(nil)).Once()
	expectedConn.On("Close").Return(error(nil)).Twice()

	actualConn, err := listener.Accept()
	require.NotNil(actualConn)
	assert.NoError(err)
	assert.Equal(1.0, gauge.Value())

	assert.Nil(actualConn.Close())
	assert.Zero(gauge.Value())

	// the gauge decrement should be idempotent
	assert.Nil(actualConn.Close())
	assert.Zero(gauge.Value())

	delegate.AssertExpectations(t)
	expectedConn.AssertExpectations(t)
}

func testInstrumentListenerAcceptError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedError = errors.New("expected error from Accept")
		logger        = logging.NewTestLogger(nil, t)
		gauge         = generic.NewGauge("test")
		delegate      = new(mockListener)
		listener      = InstrumentListener(logger, gauge, delegate)
	)

	require.NotNil(listener)

	delegate.On("Accept").Return(nil, expectedError).Once()

	actualConn, err := listener.Accept()
	assert.Nil(actualConn)
	assert.Error(err)
	assert.Zero(gauge.Value())

	delegate.AssertExpectations(t)
}

func testInstrumentListenerAcceptConnCloseError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		logger        = logging.NewTestLogger(nil, t)
		counter       = generic.NewGauge("test")
		delegate      = new(mockListener)
		expectedConn  = new(mockConn)
		expectedError = errors.New("expected error from conn.Close")
		listener      = InstrumentListener(logger, counter, delegate)
	)

	require.NotNil(listener)

	delegate.On("Accept").Return(expectedConn, error(nil)).Once()
	expectedConn.On("Close").Return(expectedError).Twice()

	actualConn, err := listener.Accept()
	require.NotNil(actualConn)
	assert.NoError(err)
	assert.Equal(1.0, counter.Value())

	assert.Equal(expectedError, actualConn.Close())
	assert.Zero(counter.Value())

	// the counter decrement should be idempotent
	assert.Equal(expectedError, actualConn.Close())
	assert.Zero(counter.Value())

	delegate.AssertExpectations(t)
	expectedConn.AssertExpectations(t)
}

func TestInstrumentListener(t *testing.T) {
	t.Run("Close", testInstrumentListenerClose)
	t.Run("CloseError", testInstrumentListenerCloseError)
	t.Run("Accept", testInstrumentListenerAccept)
	t.Run("AcceptError", testInstrumentListenerAcceptError)
	t.Run("AcceptConnCloseError", testInstrumentListenerAcceptConnCloseError)
}

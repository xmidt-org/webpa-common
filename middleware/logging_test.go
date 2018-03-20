package middleware

import (
	"context"
	"testing"

	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLoggingWithLoggable(t *testing.T) {
	var (
		require          = require.New(t)
		assert           = assert.New(t)
		expectedLogger   = logging.NewTestLogger(nil, t)
		loggable         = new(mockLoggable)
		expectedResponse = "expected response"

		logging = Logging(func(ctx context.Context, value interface{}) (interface{}, error) {
			assert.Equal(loggable, value)
			assert.Equal(expectedLogger, logging.GetLogger(ctx))
			return expectedResponse, nil
		})
	)

	require.NotNil(logging)

	loggable.On("Logger").Return(expectedLogger).Once()

	actual, err := logging(context.Background(), loggable)
	assert.Equal(expectedResponse, actual)
	assert.NoError(err)

	loggable.AssertExpectations(t)
}

func testLoggingWithoutLoggable(t *testing.T) {
	var (
		require = require.New(t)
		assert  = assert.New(t)

		expectedRequest  = "expected request"
		expectedResponse = "expected response"

		logging = Logging(func(ctx context.Context, value interface{}) (interface{}, error) {
			assert.Equal(expectedRequest, value)
			assert.Equal(logging.DefaultLogger(), logging.GetLogger(ctx))
			return expectedResponse, nil
		})
	)

	require.NotNil(logging)

	actual, err := logging(context.Background(), expectedRequest)
	assert.Equal(expectedResponse, actual)
	assert.NoError(err)
}

func TestLogging(t *testing.T) {
	t.Run("WithLoggable", testLoggingWithLoggable)
	t.Run("WithoutLoggable", testLoggingWithoutLoggable)
}

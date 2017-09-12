package wrpendpoint

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/logging/mocklogging"
	"github.com/Comcast/webpa-common/wrp"
	"github.com/go-kit/kit/log/level"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	var (
		assert = assert.New(t)

		request Request = &request{
			note: note{
				contents: []byte("request"),
			},
		}

		expected Response = &response{
			note: note{
				contents: []byte("response"),
			},
		}

		endpointCtx = context.WithValue(context.Background(), "foo", "bar")
		service     = new(mockService)
		endpoint    = New(service)
	)

	service.On("ServeWRP", request).Return(expected, error(nil)).Once()
	actual, err := endpoint(endpointCtx, request)
	assert.Equal(expected, actual)
	assert.NoError(err)
	assert.Equal(endpointCtx, request.Context())
	service.AssertExpectations(t)
}

func testTimeout(t *testing.T, timeout time.Duration) {
	var (
		assert = assert.New(t)

		request Request = &request{
			note: note{
				contents: []byte("request"),
			},
		}

		expected Response = &response{
			note: note{
				contents: []byte("response"),
			},
		}

		nextCalled = false
		next       = func(ctx context.Context, value interface{}) (interface{}, error) {
			nextCalled = true
			return expected, nil
		}

		middleware = Timeout(timeout)
	)

	actual, err := middleware(next)(context.Background(), request)
	assert.Equal(expected, actual)
	assert.NoError(err)

	timeoutCtx := request.Context()
	assert.NotNil(timeoutCtx.Done())
	deadline, ok := timeoutCtx.Deadline()
	assert.False(deadline.IsZero())
	assert.True(ok)
	assert.NotNil(timeoutCtx.Err())
}

func TestTimeout(t *testing.T) {
	for _, timeout := range []time.Duration{-1, 0, 15 * time.Second, 120 * time.Hour} {
		t.Run(timeout.String(), func(t *testing.T) {
			testTimeout(t, timeout)
		})
	}
}

func testLoggingSuccess(t *testing.T) {
	var (
		assert = assert.New(t)
		logger = mocklogging.New()

		wrpRequest = WrapAsRequest(context.Background(), &wrp.Message{
			Destination:     "mac:123412341234",
			TransactionUUID: "1234567890",
		})

		wrpResponse = WrapAsResponse(&wrp.Message{
			Destination:     "test",
			TransactionUUID: "0987654321",
		})

		endpoint = func(ctx context.Context, value interface{}) (interface{}, error) {
			return wrpResponse, nil
		}
	)

	mocklogging.OnLog(logger,
		level.Key(), level.InfoValue(),
		logging.MessageKey(), mocklogging.AnyValue(),
		"destination", wrpRequest.Destination(),
		"transactionID", wrpRequest.TransactionID(),
	).Return(error(nil)).Once()

	mocklogging.OnLog(logger,
		level.Key(), level.InfoValue(),
		logging.MessageKey(), mocklogging.AnyValue(),
		"destination", wrpResponse.Destination(),
		"transactionID", wrpResponse.TransactionID(),
	).Return(error(nil)).Once()

	value, err := Logging(logger)(endpoint)(context.Background(), wrpRequest)
	assert.Equal(wrpResponse, value)
	assert.NoError(err)

	logger.AssertExpectations(t)
}

func testLoggingError(t *testing.T) {
	var (
		logger = mocklogging.New()

		wrpRequest = WrapAsRequest(context.Background(), &wrp.Message{
			Destination:     "mac:123412341234",
			TransactionUUID: "1234567890",
		})

		expectedError = errors.New("expected")
		endpoint      = func(ctx context.Context, value interface{}) (interface{}, error) {
			return nil, expectedError
		}
	)

	mocklogging.OnLog(logger,
		level.Key(), level.InfoValue(),
		logging.MessageKey(), mocklogging.AnyValue(),
		"destination", wrpRequest.Destination(),
		"transactionID", wrpRequest.TransactionID(),
	).Return(error(nil)).Once()

	mocklogging.OnLog(logger,
		level.Key(), level.ErrorValue(),
		logging.MessageKey(), mocklogging.AnyValue(),
		"destination", wrpRequest.Destination(),
		"transactionID", wrpRequest.TransactionID(),
		logging.ErrorKey(), expectedError,
	).Return(error(nil)).Once()

	Logging(logger)(endpoint)(context.Background(), wrpRequest)
}

func TestLogging(t *testing.T) {
	t.Run("Success", testLoggingSuccess)
	t.Run("Error", testLoggingError)
}

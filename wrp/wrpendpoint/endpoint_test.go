package wrpendpoint

import (
	"context"
	"testing"
	"time"

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

func TestLogging(t *testing.T) {
}

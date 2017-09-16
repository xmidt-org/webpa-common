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

		expectedRequest Request = &request{
			note: note{
				contents: []byte("request"),
			},
		}

		expectedResponse Response = &response{
			note: note{
				contents: []byte("response"),
			},
		}

		expectedCtx = context.WithValue(context.Background(), "foo", "bar")
		service     = new(mockService)
		endpoint    = New(service)
	)

	service.On("ServeWRP", expectedCtx, expectedRequest).Return(expectedResponse, error(nil)).Once()
	actualResponse, err := endpoint(expectedCtx, expectedRequest)
	assert.Equal(expectedResponse, actualResponse)
	assert.NoError(err)
	service.AssertExpectations(t)
}

func TestWrap(t *testing.T) {
	var (
		assert = assert.New(t)

		expectedRequest Request = &request{
			note: note{
				contents: []byte("request"),
			},
		}

		expectedResponse Response = &response{
			note: note{
				contents: []byte("response"),
			},
		}

		expectedCtx    = context.WithValue(context.Background(), "foo", "bar")
		endpointCalled = false
		endpoint       = func(ctx context.Context, value interface{}) (interface{}, error) {
			endpointCalled = true
			assert.Equal(expectedCtx, ctx)
			assert.Equal(expectedRequest, value)
			return expectedResponse, nil
		}

		service = Wrap(endpoint)
	)

	actualResponse, err := service.ServeWRP(expectedCtx, expectedRequest)
	assert.Equal(expectedResponse, actualResponse)
	assert.NoError(err)
	assert.True(endpointCalled)
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

			deadline, ok := ctx.Deadline()
			assert.False(deadline.IsZero())
			assert.True(ok)

			assert.NotNil(ctx.Done())

			return expected, nil
		}

		middleware = Timeout(timeout)
	)

	actual, err := middleware(next)(context.Background(), request)
	assert.Equal(expected, actual)
	assert.NoError(err)
	assert.True(nextCalled)
}

func TestTimeout(t *testing.T) {
	for _, timeout := range []time.Duration{-1, 0, 15 * time.Second, 120 * time.Hour} {
		t.Run(timeout.String(), func(t *testing.T) {
			testTimeout(t, timeout)
		})
	}
}

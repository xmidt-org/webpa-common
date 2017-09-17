package middleware

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testTimeout(t *testing.T, timeout time.Duration) {
	var (
		assert = assert.New(t)

		expectedRequest  = "expected request"
		expectedResponse = "expected response"

		nextCalled = false
		next       = func(ctx context.Context, value interface{}) (interface{}, error) {
			nextCalled = true

			deadline, ok := ctx.Deadline()
			assert.False(deadline.IsZero())
			assert.True(ok)
			assert.NotNil(ctx.Done())

			return expectedResponse, nil
		}

		middleware = Timeout(timeout)
	)

	actualResponse, err := middleware(next)(context.Background(), expectedRequest)
	assert.Equal(expectedResponse, actualResponse)
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

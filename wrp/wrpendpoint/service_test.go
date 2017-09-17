package wrpendpoint

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServiceFunc(t *testing.T) {
	var (
		assert = assert.New(t)

		request Request = &request{
			note: note{
				contents: []byte("expected request"),
			},
		}

		expectedResponse Response = &response{
			note: note{
				contents: []byte("expected response"),
			},
		}

		expectedCtx = context.WithValue(context.Background(), "foo", "bar")

		serviceFuncCalled = false

		serviceFunc = ServiceFunc(func(ctx context.Context, r Request) (Response, error) {
			serviceFuncCalled = true
			assert.Equal(expectedCtx, ctx)
			return expectedResponse, nil
		})
	)

	actualResponse, err := serviceFunc.ServeWRP(expectedCtx, request)
	assert.Equal(expectedResponse, actualResponse)
	assert.NoError(err)
	assert.True(serviceFuncCalled)
}

package wrpendpoint

import (
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

		expected Response = &response{
			note: note{
				contents: []byte("expected response"),
			},
		}

		serviceFuncCalled = false

		serviceFunc = ServiceFunc(func(r Request) (Response, error) {
			serviceFuncCalled = true
			return expected, nil
		})
	)

	actual, err := serviceFunc.ServeWRP(request)
	assert.Equal(expected, actual)
	assert.NoError(err)
	assert.True(serviceFuncCalled)
}

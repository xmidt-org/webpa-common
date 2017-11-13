package fanout

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/go-kit/kit/endpoint"
	"github.com/stretchr/testify/assert"
)

func testComponentsApply(t *testing.T, count int) {
	var (
		assert = assert.New(t)

		middlewareCalled = false
		middleware       = func(e endpoint.Endpoint) endpoint.Endpoint {
			return func(ctx context.Context, v interface{}) (interface{}, error) {
				middlewareCalled = true
				return e(ctx, v)
			}
		}

		original = make(Components)
	)

	for repeat := 0; repeat < count; repeat++ {
		key := fmt.Sprintf("component-%d", repeat)
		original[key] = func(ctx context.Context, v interface{}) (interface{}, error) {
			return key, errors.New(key)
		}
	}

	decorated := original.Apply(middleware)
	assert.Equal(len(original), len(decorated))

	for key, endpoint := range decorated {
		_, ok := original[key]
		assert.True(ok)

		middlewareCalled = false
		response, err := endpoint(context.Background(), struct{}{})
		assert.Equal(key, response)
		assert.Equal(errors.New(key), err)
		assert.True(middlewareCalled)
	}
}

func TestComponents(t *testing.T) {
	t.Run("Apply", func(t *testing.T) {
		for _, count := range []int{0, 1, 3} {
			t.Run(fmt.Sprintf("Len=%d", count), func(t *testing.T) {
				testComponentsApply(t, count)
			})
		}
	})
}

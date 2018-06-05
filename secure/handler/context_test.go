package handler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFromContextSatID(t *testing.T) {
	t.Run("NoSatID", func(t *testing.T) {
		assert := assert.New(t)
		val, ofType := FromContext(context.Background())

		assert.False(ofType)
		assert.Nil(val)
	})

	t.Run("PresentSatID", func(t *testing.T) {
		assert := assert.New(t)
		inputCtxValues := &ContextValues{
			SatClientID: "test",
			Path:        "foo",
			Method:      "GET",
		}
		inputContext := context.WithValue(context.Background(), contextKey{}, inputCtxValues)
		val, ofType := FromContext(inputContext)

		assert.True(ofType)
		assert.Equal(inputCtxValues, val)
	})
}

func TestNewContext(t *testing.T) {
	assert := assert.New(t)
	inputCtxValues := &ContextValues{
		SatClientID: "test",
		Path:        "foo",
		Method:      "GET",
	}
	expectedContext := context.WithValue(context.Background(), contextKey{}, inputCtxValues)
	actualContext := NewContextWithValue(context.Background(), inputCtxValues)

	assert.EqualValues(expectedContext, actualContext)
}

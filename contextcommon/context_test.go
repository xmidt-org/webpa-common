package contextcommon

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type genericContextKey struct{}

var genericKey = genericContextKey{}

func TestFromContextSatID(t *testing.T) {
	t.Run("NoSatID", func(t *testing.T) {
		assert := assert.New(t)
		val := FromContext(context.Background(), genericKey)

		assert.Nil(val)
	})

	t.Run("PresentSatID", func(t *testing.T) {
		assert := assert.New(t)
		inputContext := context.WithValue(context.Background(), genericKey, "test")
		val := FromContext(inputContext, genericKey)
		valStr, ok := val.(string)

		assert.True(ok)
		assert.EqualValues("test", valStr)
	})
}

func TestNewContext(t *testing.T) {
	assert := assert.New(t)
	expectedContext := context.WithValue(context.Background(), genericKey, "test")
	actualContext := NewContextWithValue(context.Background(), genericKey, "test")

	assert.EqualValues(expectedContext, actualContext)
}

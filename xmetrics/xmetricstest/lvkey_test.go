package xmetricstest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLVKey(t *testing.T) {
	t.Run("Root", func(t *testing.T) {
		assert := assert.New(t)
		assert.True(rootKey.Root())
		assert.True(LVKey("").Root())
		assert.False(LVKey("askldfasdkfjaskdjf").Root())
	})
}

func testNewLVKeyRoot(t *testing.T, labelsAndValues []string) {
	var (
		assert   = assert.New(t)
		require  = require.New(t)
		key, err = NewLVKey(labelsAndValues)
	)

	assert.NoError(err)
	require.NotNil(key)
	assert.True(key.Root())
}

func testNewLVKeyInvalid(t *testing.T, labelsAndValues []string) {
	var (
		assert   = assert.New(t)
		require  = require.New(t)
		key, err = NewLVKey(labelsAndValues)
	)

	assert.Error(err)
	require.NotNil(key)
	assert.True(key.Root())
}

func testNewLVKeySuccess(t *testing.T, labelsAndValues []string, expected LVKey) {
	var (
		assert      = assert.New(t)
		require     = require.New(t)
		actual, err = NewLVKey(labelsAndValues)
	)

	assert.NoError(err)
	require.NotNil(actual)
	assert.False(actual.Root())
	assert.Equal(expected, actual)
}

func TestNewLVKey(t *testing.T) {
	t.Run("Root", func(t *testing.T) {
		testNewLVKeyRoot(t, nil)
		testNewLVKeyRoot(t, []string{})
	})

	t.Run("Invalid", func(t *testing.T) {
		testNewLVKeyInvalid(t, []string{"one"})
		testNewLVKeyInvalid(t, []string{"one", "two", "three"})
		testNewLVKeyInvalid(t, []string{"one", "two", "three", "four", "five"})
	})

	t.Run("Success", func(t *testing.T) {
		testNewLVKeySuccess(t, []string{"code", "500"}, "code=500")
		testNewLVKeySuccess(t, []string{"code", "500", "method", "POST"}, "code=500,method=POST")
		testNewLVKeySuccess(t, []string{"method", "POST", "code", "500"}, "code=500,method=POST")
		testNewLVKeySuccess(t, []string{"code", "500", "event", "notify", "method", "POST"}, "code=500,event=notify,method=POST")
		testNewLVKeySuccess(t, []string{"code", "500", "method", "POST", "event", "notify"}, "code=500,event=notify,method=POST")
		testNewLVKeySuccess(t, []string{"method", "POST", "event", "notify", "code", "500"}, "code=500,event=notify,method=POST")
	})
}

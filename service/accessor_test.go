package service

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmptyAccessor(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		ea      = EmptyAccessor()
	)

	require.NotNil(ea)
	i, err := ea.Get([]byte("does not matter"))
	assert.Empty(i)
	assert.Error(err)
}

func TestMapAccessor(t *testing.T) {
	var (
		assert = assert.New(t)
		ma     = MapAccessor{"test": "a valid instance"}
	)

	i, err := ma.Get([]byte("test"))
	assert.Equal("a valid instance", i)
	assert.NoError(err)

	i, err = ma.Get([]byte("nosuch"))
	assert.Empty(i)
	assert.Error(err)
}

func TestUpdatableAccessor(t *testing.T) {
	var (
		assert = assert.New(t)
		ua     = new(UpdatableAccessor)
	)

	i, err := ua.Get([]byte("test"))
	assert.Empty(i)
	assert.Error(err)

	ua.SetInstances(MapAccessor{"test": "a valid instance"})
	i, err = ua.Get([]byte("test"))
	assert.Equal("a valid instance", i)
	assert.NoError(err)
	i, err = ua.Get([]byte("nosuch"))
	assert.Empty(i)
	assert.Error(err)

	ua.SetInstances(EmptyAccessor())
	i, err = ua.Get([]byte("test"))
	assert.Empty(i)
	assert.Error(err)
	i, err = ua.Get([]byte("nosuch"))
	assert.Empty(i)
	assert.Error(err)

	expectedError := errors.New("expected 1")
	ua.SetError(expectedError)
	i, err = ua.Get([]byte("test"))
	assert.Empty(i)
	assert.Equal(expectedError, err)
	i, err = ua.Get([]byte("nosuch"))
	assert.Empty(i)
	assert.Equal(expectedError, err)

	ua.Update(MapAccessor{"test": "a valid instance"}, nil)
	i, err = ua.Get([]byte("test"))
	assert.Equal("a valid instance", i)
	assert.NoError(err)
	i, err = ua.Get([]byte("nosuch"))
	assert.Empty(i)
	assert.Error(err)

	expectedError = errors.New("expected 2")
	ua.Update(MapAccessor{"test": "a valid instance"}, expectedError)
	i, err = ua.Get([]byte("test"))
	assert.Empty(i)
	assert.Equal(expectedError, err)
	i, err = ua.Get([]byte("nosuch"))
	assert.Empty(i)
	assert.Equal(expectedError, err)

	expectedError = errors.New("expected 3")
	ua.Update(nil, expectedError)
	i, err = ua.Get([]byte("test"))
	assert.Empty(i)
	assert.Equal(expectedError, err)
	i, err = ua.Get([]byte("nosuch"))
	assert.Empty(i)
	assert.Equal(expectedError, err)
}

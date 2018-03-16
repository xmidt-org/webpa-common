package service

import (
	"testing"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/service/servicemock"
	"github.com/go-kit/kit/sd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testNewContextualInstancerEmpty(t *testing.T, m map[string]interface{}) {
	var (
		assert = assert.New(t)
		next   = new(servicemock.Instancer)
	)

	i := NewContextualInstancer(next, m)
	assert.Equal(next, i)
	_, ok := i.(logging.Contextual)
	assert.False(ok)

	next.AssertExpectations(t)
}

func testNewContextualInstancerWithMetadata(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		next = new(servicemock.Instancer)
		m    = map[string]interface{}{"key": "value"}
	)

	i := NewContextualInstancer(next, m)
	require.NotNil(i)
	assert.NotEqual(next, i)

	c, ok := i.(logging.Contextual)
	require.True(ok)
	require.NotNil(c)

	assert.Equal(m, c.Metadata())
}

func TestNewContextualInstancer(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		testNewContextualInstancerEmpty(t, map[string]interface{}{})
	})

	t.Run("Nil", func(t *testing.T) {
		testNewContextualInstancerEmpty(t, nil)
	})

	t.Run("WithMetadata", testNewContextualInstancerWithMetadata)
}

func testInstancers(t *testing.T, is Instancers) {
	assert := assert.New(t)

	assert.Equal(0, is.Len())
	assert.False(is.Has("nosuch"))
	i, ok := is.Get("nosuch")
	assert.Nil(i)
	assert.False(ok)

	assert.NotPanics(func() { is.Stop() })

	var (
		child1 = new(servicemock.Instancer)
		child2 = new(servicemock.Instancer)
	)

	is.Set("child1", child1)
	assert.Equal(1, is.Len())

	assert.False(is.Has("nosuch"))
	i, ok = is.Get("nosuch")
	assert.Nil(i)
	assert.False(ok)

	i, ok = is.Get("child1")
	assert.Equal(child1, i)
	assert.True(ok)

	is.Set("child2", child2)
	assert.Equal(2, is.Len())

	assert.False(is.Has("nosuch"))
	i, ok = is.Get("nosuch")
	assert.Nil(i)
	assert.False(ok)

	i, ok = is.Get("child1")
	assert.Equal(child1, i)
	assert.True(ok)

	i, ok = is.Get("child2")
	assert.Equal(child2, i)
	assert.True(ok)

	child1.On("Stop").Once()
	child2.On("Stop").Once()
	assert.NotPanics(func() { is.Stop() })

	assert.Equal(
		map[string]sd.Instancer{
			"child1": child1,
			"child2": child2,
		},
		map[string]sd.Instancer(is.Copy()),
	)

	child1.AssertExpectations(t)
	child2.AssertExpectations(t)
}

func TestInstancers(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		testInstancers(t, Instancers{})
		testInstancers(t, Instancers{}.Copy())
	})

	t.Run("Nil", func(t *testing.T) {
		testInstancers(t, nil)
		testInstancers(t, Instancers(nil).Copy())
	})
}

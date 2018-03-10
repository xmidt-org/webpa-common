package service

import (
	"testing"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/service/servicemock"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testInstancers(t *testing.T, is Instancers) {
	assert := assert.New(t)

	assert.Equal(0, is.Len())
	assert.False(is.Has("nosuch"))
	l, i, ok := is.Get("nosuch")
	assert.Nil(l)
	assert.Nil(i)
	assert.False(ok)

	is.Each(func(string, log.Logger, sd.Instancer) {
		assert.Fail("The predicate should not have been called for an empty Instancers")
	})

	assert.NotPanics(func() { is.Stop() })

	var (
		logger = logging.NewTestLogger(nil, t)
		child1 = new(servicemock.Instancer)
		child2 = new(servicemock.Instancer)
	)

	is.Set("child1", logger, child1)
	assert.Equal(1, is.Len())

	assert.False(is.Has("nosuch"))
	l, i, ok = is.Get("nosuch")
	assert.Nil(l)
	assert.Nil(i)
	assert.False(ok)

	l, i, ok = is.Get("child1")
	assert.Equal(logger, l)
	assert.Equal(child1, i)
	assert.True(ok)

	eachCount := 0
	is.Each(func(key string, l log.Logger, i sd.Instancer) {
		assert.Equal(0, eachCount, "The predicate should be called only once")
		eachCount++
		assert.Equal("child1", key)
		assert.Equal(logger, l)
		assert.Equal(child1, i)
	})

	is.Set("child2", nil, child2)
	assert.Equal(2, is.Len())

	assert.False(is.Has("nosuch"))
	l, i, ok = is.Get("nosuch")
	assert.Nil(l)
	assert.Nil(i)
	assert.False(ok)

	l, i, ok = is.Get("child1")
	assert.Equal(logger, l)
	assert.Equal(child1, i)
	assert.True(ok)

	l, i, ok = is.Get("child2")
	assert.Equal(logging.DefaultLogger(), l)
	assert.Equal(child2, i)
	assert.True(ok)

	visitedKeys := make(map[string]bool)
	is.Each(func(key string, l log.Logger, i sd.Instancer) {
		switch key {
		case "child1":
			assert.Equal(logger, l)
			assert.Equal(child1, i)
			visitedKeys["child1"] = true
		case "child2":
			assert.Equal(logging.DefaultLogger(), l)
			assert.Equal(child2, i)
			visitedKeys["child2"] = true
		default:
			assert.Fail("The predicate should only be called for keys in the map")
		}
	})

	assert.Equal(map[string]bool{"child1": true, "child2": true}, visitedKeys)

	child1.On("Stop").Once()
	child2.On("Stop").Once()
	assert.NotPanics(func() { is.Stop() })

	child1.AssertExpectations(t)
	child2.AssertExpectations(t)

	clone := is.Copy()
	assert.Equal(2, clone.Len())

	assert.False(clone.Has("nosuch"))
	l, i, ok = clone.Get("nosuch")
	assert.Nil(l)
	assert.Nil(i)
	assert.False(ok)

	l, i, ok = clone.Get("child1")
	assert.Equal(logger, l)
	assert.Equal(child1, i)
	assert.True(ok)

	l, i, ok = clone.Get("child2")
	assert.Equal(logging.DefaultLogger(), l)
	assert.Equal(child2, i)
	assert.True(ok)
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

func TestNewFixedInstancers(t *testing.T) {
	var (
		assert            = assert.New(t)
		require           = require.New(t)
		logger            = logging.NewTestLogger(nil, t)
		expectedInstances = []string{"instance1", "instance2"}
	)

	is := NewFixedInstancers(logger, "test", expectedInstances)
	require.Equal(1, is.Len())
	l, i, ok := is.Get("test")
	assert.NotNil(l)
	assert.Equal(sd.FixedInstancer(expectedInstances), i)
	assert.True(ok)

	is = NewFixedInstancers(nil, "test", expectedInstances)
	require.Equal(1, is.Len())
	l, i, ok = is.Get("test")
	assert.NotNil(l)
	assert.Equal(sd.FixedInstancer(expectedInstances), i)
	assert.True(ok)
}

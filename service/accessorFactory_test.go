package service

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testNewConsistentAccessorEmpty(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
	)

	for _, i := range [][]string{nil, []string{}} {
		a := newConsistentAccessor(111, i)
		require.NotNil(a)
		i, err := a.Get([]byte("test"))
		assert.Empty(i)
		assert.Error(err)
	}
}

func testNewConsistentAccessorNonEmpty(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		a = newConsistentAccessor(123, []string{"an instance"})
	)

	require.NotNil(a)
	for _, k := range []string{"a", "alsdkjfa;lksehjuro8iwurjhf", "asdf8974", "875kjh4", "928375hjdfgkyu9832745kjshdfgoi873465"} {
		i, err := a.Get([]byte(k))
		assert.Equal("an instance", i)
		assert.NoError(err)
	}
}

func TestNewConsistentAccessor(t *testing.T) {
	t.Run("Empty", testNewConsistentAccessorEmpty)
	t.Run("Nonempty", testNewConsistentAccessorNonEmpty)
}

func testNewConsistentAccessorFactory(t *testing.T, vnodeCount int) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		af = NewConsistentAccessorFactory(vnodeCount)
	)

	require.NotNil(af)
	a := af([]string{"an instance"})
	require.NotNil(a)
	for _, k := range []string{"a", "alsdkjfa;lksehjuro8iwurjhf", "asdf8974", "875kjh4", "928375hjdfgkyu9832745kjshdfgoi873465"} {
		i, err := a.Get([]byte(k))
		assert.Equal("an instance", i)
		assert.NoError(err)
	}
}

func TestNewConsistentAccessorFactory(t *testing.T) {
	for _, v := range []int{-1, 0, 123, DefaultVnodeCount, 756} {
		t.Run(fmt.Sprintf("vnodeCount=%d", v), func(t *testing.T) {
			testNewConsistentAccessorFactory(t, v)
		})
	}
}

func TestDefaultAccessorFactory(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		a = DefaultAccessorFactory([]string{"an instance"})
	)

	require.NotNil(a)
	for _, k := range []string{"a", "alsdkjfa;lksehjuro8iwurjhf", "asdf8974", "875kjh4", "928375hjdfgkyu9832745kjshdfgoi873465"} {
		i, err := a.Get([]byte(k))
		assert.Equal("an instance", i)
		assert.NoError(err)
	}
}

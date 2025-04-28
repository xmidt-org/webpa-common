package accessor

import (
	"fmt"
	"testing"

	"github.com/xmidt-org/webpa-common/v2/xhttp/gate"

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

		a = newConsistentAccessor(123, []string{"https://example.com"})
	)

	require.NotNil(a)
	for _, k := range []string{"a", "alsdkjfa;lksehjuro8iwurjhf", "asdf8974", "875kjh4", "928375hjdfgkyu9832745kjshdfgoi873465"} {
		i, err := a.Get([]byte(k))
		assert.Equal("https://example.com", i)
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
	a := af([]string{"https://example.com"})
	require.NotNil(a)
	for _, k := range []string{"a", "alsdkjfa;lksehjuro8iwurjhf", "asdf8974", "875kjh4", "928375hjdfgkyu9832745kjshdfgoi873465"} {
		i, err := a.Get([]byte(k))
		assert.Equal("https://example.com", i)
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

func testNewConsistentAccessorFactoryWithGate(t *testing.T, vnodeCount int, g gate.Interface) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		af = NewConsistentAccessorFactoryWithGate(vnodeCount, g)
	)

	require.NotNil(af)
	a := af([]string{"https://example.com"})
	require.NotNil(a)
	for _, k := range []string{"a", "alsdkjfa;lksehjuro8iwurjhf", "asdf8974", "875kjh4", "928375hjdfgkyu9832745kjshdfgoi873465"} {
		i, err := a.Get([]byte(k))
		assert.Equal("https://example.com", i)
		if (g != nil && g.Open()) || g == nil {
			assert.NoError(err)
		} else if g != nil && !g.Open() {
			assert.Error(err)
		}
	}
}

func TestNewConsistentAccessorFactoryWithGate(t *testing.T) {
	for _, v := range []int{-1, 0, 123, DefaultVnodeCount, 756} {
		t.Run(fmt.Sprintf("vnodeCount=%d", v), func(t *testing.T) {
			t.Run("NilGate", func(t *testing.T) {
				testNewConsistentAccessorFactoryWithGate(t, v, nil)
			})
			t.Run("GateUp", func(t *testing.T) {
				testNewConsistentAccessorFactoryWithGate(t, v, gate.New(true))
			})
			t.Run("GateDown", func(t *testing.T) {
				testNewConsistentAccessorFactoryWithGate(t, v, gate.New(false))
			})
		})
	}
}

func TestDefaultAccessorFactory(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		a = DefaultAccessorFactory([]string{"https://example.com"})
	)

	require.NotNil(a)
	for _, k := range []string{"a", "alsdkjfa;lksehjuro8iwurjhf", "asdf8974", "875kjh4", "928375hjdfgkyu9832745kjshdfgoi873465"} {
		i, err := a.Get([]byte(k))
		assert.Equal("https://example.com", i)
		assert.NoError(err)
	}
}

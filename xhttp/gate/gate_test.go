package gate

import (
	"testing"

	"github.com/go-kit/kit/metrics/generic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testNewBadInitialState(t *testing.T) {
	assert := assert.New(t)
	assert.Panics(func() {
		New(uint32(1234123))
	})
}

func testNewString(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		g1 = New(Open)
		g2 = New(Closed)
	)

	require.NotNil(g1)
	require.NotNil(g2)
	assert.NotEqual(g2.String(), g1.String())
}

func testNewInitiallyOpen(t *testing.T, g Interface) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
	)

	require.NotNil(g)
	assert.True(g.Open())

	assert.False(g.Raise())
	assert.True(g.Open())

	assert.True(g.Lower())
	assert.False(g.Open())

	assert.False(g.Lower())
	assert.False(g.Open())

	assert.True(g.Raise())
	assert.True(g.Open())
}

func testNewInitiallyClosed(t *testing.T, g Interface) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
	)

	require.NotNil(g)
	assert.False(g.Open())

	assert.False(g.Lower())
	assert.False(g.Open())

	assert.True(g.Raise())
	assert.True(g.Open())

	assert.False(g.Raise())
	assert.True(g.Open())

	assert.True(g.Lower())
	assert.False(g.Open())
}

func testNewInitiallyOpenWithGauge(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		gauge = generic.NewGauge("test")
		g     = New(Open, WithGauge(gauge))
	)

	require.NotNil(g)
	assert.True(g.Open())
	assert.Equal(GaugeOpen, gauge.Value())

	assert.False(g.Raise())
	assert.True(g.Open())
	assert.Equal(GaugeOpen, gauge.Value())

	assert.True(g.Lower())
	assert.False(g.Open())
	assert.Equal(GaugeClosed, gauge.Value())

	assert.False(g.Lower())
	assert.False(g.Open())
	assert.Equal(GaugeClosed, gauge.Value())

	assert.True(g.Raise())
	assert.True(g.Open())
	assert.Equal(GaugeOpen, gauge.Value())
}

func testNewInitiallyClosedWithGauge(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		gauge = generic.NewGauge("test")
		g     = New(Closed, WithGauge(gauge))
	)

	require.NotNil(g)
	assert.False(g.Open())
	assert.Equal(GaugeClosed, gauge.Value())

	assert.False(g.Lower())
	assert.False(g.Open())
	assert.Equal(GaugeClosed, gauge.Value())

	assert.True(g.Raise())
	assert.True(g.Open())
	assert.Equal(GaugeOpen, gauge.Value())

	assert.False(g.Raise())
	assert.True(g.Open())
	assert.Equal(GaugeOpen, gauge.Value())

	assert.True(g.Lower())
	assert.False(g.Open())
	assert.Equal(GaugeClosed, gauge.Value())
}

func TestNew(t *testing.T) {
	t.Run("BadInitialState", testNewBadInitialState)
	t.Run("String", testNewString)

	t.Run("InitiallyOpen", func(t *testing.T) {
		testNewInitiallyOpen(t, New(Open))
		testNewInitiallyOpen(t, New(Open, WithGauge(nil)))

		t.Run("WithGauge", testNewInitiallyOpenWithGauge)
	})

	t.Run("InitiallyClosed", func(t *testing.T) {
		testNewInitiallyClosed(t, New(Closed))
		testNewInitiallyClosed(t, New(Closed, WithGauge(nil)))

		t.Run("WithGauge", testNewInitiallyClosedWithGauge)
	})
}

package gate

import (
	"testing"

	"github.com/Comcast/webpa-common/xmetrics/xmetricstest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testNewDefault(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
	)

	g := New()
	require.NotNil(g)
	assert.True(g.IsOpen())

	g.Raise()
	assert.True(g.IsOpen())

	g.Lower()
	assert.False(g.IsOpen())

	g.Lower()
	assert.False(g.IsOpen())

	g.Raise()
	assert.True(g.IsOpen())
}

func testNewNilClosedGauge(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
	)

	g := New(WithClosedGauge(nil))
	require.NotNil(g)
	assert.True(g.IsOpen())

	g.Raise()
	assert.True(g.IsOpen())

	g.Lower()
	assert.False(g.IsOpen())

	g.Lower()
	assert.False(g.IsOpen())

	g.Raise()
	assert.True(g.IsOpen())
}

func testNewCustomInitiallyOpen(t *testing.T) {
	var (
		assert      = assert.New(t)
		require     = require.New(t)
		provider    = xmetricstest.NewProvider(nil)
		closedGauge = provider.NewGauge("test")
	)

	g := New(WithClosedGauge(closedGauge))
	require.NotNil(g)
	assert.True(g.IsOpen())
	provider.Assert(t, "test")(xmetricstest.Value(0.0))

	g.Raise()
	assert.True(g.IsOpen())
	provider.Assert(t, "test")(xmetricstest.Value(0.0))

	g.Lower()
	assert.False(g.IsOpen())
	provider.Assert(t, "test")(xmetricstest.Value(1.0))

	g.Lower()
	assert.False(g.IsOpen())
	provider.Assert(t, "test")(xmetricstest.Value(1.0))

	g.Raise()
	assert.True(g.IsOpen())
	provider.Assert(t, "test")(xmetricstest.Value(0.0))
}

func testNewCustomInitiallyClosed(t *testing.T) {
	var (
		assert      = assert.New(t)
		require     = require.New(t)
		provider    = xmetricstest.NewProvider(nil)
		closedGauge = provider.NewGauge("test")
	)

	g := New(WithInitiallyClosed(), WithClosedGauge(closedGauge))
	require.NotNil(g)
	assert.False(g.IsOpen())
	provider.Assert(t, "test")(xmetricstest.Value(1.0))

	g.Lower()
	assert.False(g.IsOpen())
	provider.Assert(t, "test")(xmetricstest.Value(1.0))

	g.Raise()
	assert.True(g.IsOpen())
	provider.Assert(t, "test")(xmetricstest.Value(0.0))

	g.Raise()
	assert.True(g.IsOpen())
	provider.Assert(t, "test")(xmetricstest.Value(0.0))

	g.Lower()
	assert.False(g.IsOpen())
	provider.Assert(t, "test")(xmetricstest.Value(1.0))
}

func TestNew(t *testing.T) {
	t.Run("Default", testNewDefault)
	t.Run("NilGauge", testNewNilClosedGauge)
	t.Run("Custom", testNewCustomInitiallyOpen)
	t.Run("Custom", testNewCustomInitiallyClosed)
}

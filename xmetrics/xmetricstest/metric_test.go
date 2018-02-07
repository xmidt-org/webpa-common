package xmetricstest

import (
	"testing"

	"github.com/Comcast/webpa-common/xmetrics"
	"github.com/go-kit/kit/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCounter(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		c = NewCounter("test")
	)

	require.NotNil(c)
	require.Implements((*xmetrics.Valuer)(nil), c)
	require.Implements((*Labeled)(nil), c)

	assert.Panics(func() {
		c.With("one")
	})

	c.Add(1.0)
	assert.Equal(1.0, c.(xmetrics.Valuer).Value())

	{
		nosuch, ok := c.(Labeled).Get("code=500")
		assert.Nil(nosuch)
		assert.False(ok)
	}

	child1 := c.With("code", "500")
	require.NotNil(child1)
	require.Implements((*xmetrics.Valuer)(nil), child1)
	child1.Add(2.0)
	assert.Equal(1.0, c.(xmetrics.Valuer).Value())
	assert.Equal(2.0, child1.(xmetrics.Valuer).Value())

	assert.True(child1 == child1.With("code", "500"))
	assert.True(child1 == c.With("code", "500"))

	{
		child2, ok := c.(Labeled).Get("code=500")
		assert.True(child1 == child2)
		assert.True(ok)
	}
}

func TestGauge(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		g = NewGauge("test")
	)

	require.NotNil(g)
	require.Implements((*xmetrics.Valuer)(nil), g)
	require.Implements((*Labeled)(nil), g)

	assert.Panics(func() {
		g.With("one")
	})

	g.Add(1.0)
	assert.Equal(1.0, g.(xmetrics.Valuer).Value())

	g.Set(15.0)
	assert.Equal(15.0, g.(xmetrics.Valuer).Value())

	{
		nosuch, ok := g.(Labeled).Get("code=500")
		assert.Nil(nosuch)
		assert.False(ok)
	}

	child1 := g.With("code", "500")
	require.NotNil(child1)
	require.Implements((*xmetrics.Valuer)(nil), child1)
	child1.Add(2.0)
	assert.Equal(15.0, g.(xmetrics.Valuer).Value())
	assert.Equal(2.0, child1.(xmetrics.Valuer).Value())

	child1.Set(17.5)
	assert.Equal(15.0, g.(xmetrics.Valuer).Value())
	assert.Equal(17.5, child1.(xmetrics.Valuer).Value())

	assert.True(child1 == child1.With("code", "500"))
	assert.True(child1 == g.With("code", "500"))

	{
		child2, ok := g.(Labeled).Get("code=500")
		assert.True(child1 == child2)
		assert.True(ok)
	}
}

func TestHistogram(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		h = NewHistogram("test", 5)
	)

	require.NotNil(h)
	require.Implements((*Labeled)(nil), h)

	assert.Panics(func() {
		h.With("one")
	})

	h.Observe(1.0)

	{
		nosuch, ok := h.(Labeled).Get("code=500")
		assert.Nil(nosuch)
		assert.False(ok)
	}

	child1 := h.With("code", "500")
	require.NotNil(child1)
	child1.Observe(2.0)

	assert.True(child1 == child1.With("code", "500"))
	assert.True(child1 == h.With("code", "500"))

	{
		child2, ok := h.(Labeled).Get("code=500")
		assert.True(child1 == child2)
		assert.True(ok)
	}
}

func testNewMetricMissingName(t *testing.T) {
	assert := assert.New(t)
	c, err := NewMetric(xmetrics.Metric{Type: "counter"})
	assert.Nil(c)
	assert.Error(err)
}

func testNewMetricUnsupportedType(t *testing.T) {
	assert := assert.New(t)
	c, err := NewMetric(xmetrics.Metric{Name: "test", Type: "unsupported"})
	assert.Nil(c)
	assert.Error(err)
}

func testNewMetricCounter(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		c, err  = NewMetric(xmetrics.Metric{Name: "test", Type: "counter"})
	)

	require.NotNil(c)
	require.NoError(err)

	assert.Implements((*metrics.Counter)(nil), c)
	assert.Implements((*Labeled)(nil), c)
	assert.Implements((*xmetrics.Valuer)(nil), c)
}

func testNewMetricGauge(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		c, err  = NewMetric(xmetrics.Metric{Name: "test", Type: "gauge"})
	)

	require.NotNil(c)
	require.NoError(err)

	assert.Implements((*metrics.Gauge)(nil), c)
	assert.Implements((*Labeled)(nil), c)
	assert.Implements((*xmetrics.Valuer)(nil), c)
}

func testNewMetricHistogram(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		c, err  = NewMetric(xmetrics.Metric{Name: "test", Type: "histogram"})
	)

	require.NotNil(c)
	require.NoError(err)

	assert.Implements((*metrics.Histogram)(nil), c)
	assert.Implements((*Labeled)(nil), c)
}

func testNewMetricSummary(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		c, err  = NewMetric(xmetrics.Metric{Name: "test", Type: "summary"})
	)

	require.NotNil(c)
	require.NoError(err)

	assert.Implements((*metrics.Histogram)(nil), c)
	assert.Implements((*Labeled)(nil), c)
}

func TestNewMetric(t *testing.T) {
	t.Run("MissingName", testNewMetricMissingName)
	t.Run("UnsupportedType", testNewMetricUnsupportedType)
	t.Run("Counter", testNewMetricCounter)
	t.Run("Gauge", testNewMetricGauge)
	t.Run("Histogram", testNewMetricHistogram)
	t.Run("Summary", testNewMetricSummary)
}

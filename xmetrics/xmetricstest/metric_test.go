package xmetricstest

import (
	"testing"

	"github.com/Comcast/webpa-common/xmetrics"
	"github.com/go-kit/kit/metrics/generic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	_, ok := c.(*generic.Counter)
	assert.True(ok)
}

func testNewMetricGauge(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		c, err  = NewMetric(xmetrics.Metric{Name: "test", Type: "gauge"})
	)

	require.NotNil(c)
	require.NoError(err)

	_, ok := c.(*generic.Gauge)
	assert.True(ok)
}

func testNewMetricHistogram(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		c, err  = NewMetric(xmetrics.Metric{Name: "test", Type: "histogram"})
	)

	require.NotNil(c)
	require.NoError(err)

	_, ok := c.(*generic.Histogram)
	assert.True(ok)
}

func testNewMetricSummary(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		c, err  = NewMetric(xmetrics.Metric{Name: "test", Type: "summary"})
	)

	require.NotNil(c)
	require.NoError(err)

	_, ok := c.(*generic.Histogram)
	assert.True(ok)
}

func TestNewMetric(t *testing.T) {
	t.Run("MissingName", testNewMetricMissingName)
	t.Run("UnsupportedType", testNewMetricUnsupportedType)
	t.Run("Counter", testNewMetricCounter)
	t.Run("Gauge", testNewMetricGauge)
	t.Run("Histogram", testNewMetricHistogram)
	t.Run("Summary", testNewMetricSummary)
}

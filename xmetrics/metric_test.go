package xmetrics

import (
	"testing"

	"github.com/Comcast/webpa-common/logging"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testNewCollectorMissingName(t *testing.T) {
	assert := assert.New(t)
	c, err := NewCollector(Metric{Type: "counter"})
	assert.Nil(c)
	assert.Error(err)
}

func testNewCollectorUnsupportedType(t *testing.T) {
	assert := assert.New(t)
	c, err := NewCollector(Metric{Name: "test", Type: "unsupported"})
	assert.Nil(c)
	assert.Error(err)
}

func testNewCollectorCounter(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		c, err  = NewCollector(Metric{Name: "test", Type: "counter"})
	)

	require.NotNil(c)
	require.NoError(err)

	_, ok := c.(*prometheus.CounterVec)
	assert.True(ok)
}

func testNewCollectorGauge(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		c, err  = NewCollector(Metric{Name: "test", Type: "gauge"})
	)

	require.NotNil(c)
	require.NoError(err)

	_, ok := c.(*prometheus.GaugeVec)
	assert.True(ok)
}

func testNewCollectorHistogram(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		c, err  = NewCollector(Metric{Name: "test", Type: "histogram"})
	)

	require.NotNil(c)
	require.NoError(err)

	_, ok := c.(*prometheus.HistogramVec)
	assert.True(ok)
}

func testNewCollectorSummary(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		c, err  = NewCollector(Metric{Name: "test", Type: "summary"})
	)

	require.NotNil(c)
	require.NoError(err)

	_, ok := c.(*prometheus.SummaryVec)
	assert.True(ok)
}

func TestNewCollector(t *testing.T) {
	t.Run("MissingName", testNewCollectorMissingName)
	t.Run("UnsupportedType", testNewCollectorUnsupportedType)
	t.Run("Counter", testNewCollectorCounter)
	t.Run("Gauge", testNewCollectorGauge)
	t.Run("Histogram", testNewCollectorHistogram)
	t.Run("Summary", testNewCollectorSummary)
}

func TestMerger(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			assert = assert.New(t)
			merger = NewMerger().
				Logger(logging.NewTestLogger(nil, t)).
				AddMetrics(false, []Metric{{Name: "counter", Type: "counter"}}).
				AddModules(true, func() []Metric { return []Metric{{Name: "counter", Type: "counter"}} })
		)

		assert.Len(merger.Merged(), 1)
		assert.NoError(merger.Err())
	})
}

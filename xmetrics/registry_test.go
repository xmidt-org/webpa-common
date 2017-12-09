package xmetrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRegistryAsGoKitProvider(t *testing.T) {
	var (
		require = require.New(t)

		o = &Options{
			Namespace: "test",
			Subsystem: "basic",
			Metrics: map[string]Metric{
				"counter": Metric{
					Type: "counter",
					Help: "a test counter",
				},
				"gauge": Metric{
					Type: "gauge",
					Help: "a test gauge",
				},
				"histogram": Metric{
					Type:    "histogram",
					Buckets: []float64{0.5, 1.0, 1.5},
				},
				"summary": Metric{
					Type:   "summary",
					MaxAge: 15 * time.Hour,
				},
			},
		}
	)

	r, err := NewRegistry(o)
	require.NoError(err)
	require.NotNil(r)

	t.Run("NewCounter", func(t *testing.T) {
		assert := assert.New(t)
		preregistered := r.NewCounter("counter")
		assert.NotNil(preregistered)
		assert.Equal(preregistered, r.NewCounter("counter"))

		adHoc := r.NewCounter("new_counter")
		assert.NotNil(adHoc)
		assert.NotEqual(preregistered, adHoc)
		assert.Equal(adHoc, r.NewCounter("new_counter"))

		assert.Panics(func() { r.NewCounter("gauge") })
		assert.Panics(func() { r.NewCounter("histogram") })
		assert.Panics(func() { r.NewCounter("summary") })
	})

	t.Run("NewGauge", func(t *testing.T) {
		assert := assert.New(t)
		preregistered := r.NewGauge("gauge")
		assert.NotNil(preregistered)
		assert.Equal(preregistered, r.NewGauge("gauge"))

		adHoc := r.NewGauge("new_gauge")
		assert.NotNil(adHoc)
		assert.NotEqual(preregistered, adHoc)
		assert.Equal(adHoc, r.NewGauge("new_gauge"))

		assert.Panics(func() { r.NewGauge("counter") })
		assert.Panics(func() { r.NewGauge("histogram") })
		assert.Panics(func() { r.NewGauge("summary") })
	})

	t.Run("NewHistogram", func(t *testing.T) {
		assert := assert.New(t)
		preregistered := r.NewHistogram("histogram", 12)
		assert.NotNil(preregistered)
		assert.Equal(preregistered, r.NewHistogram("histogram", 34))

		adHoc := r.NewHistogram("new_histogram", 93)
		assert.NotNil(adHoc)
		assert.NotEqual(preregistered, adHoc)
		assert.Equal(adHoc, r.NewHistogram("new_histogram", -123))

		assert.Panics(func() { r.NewHistogram("counter", 12) })
		assert.Panics(func() { r.NewHistogram("gauge", 65344) })

		preregistered = r.NewHistogram("summary", 12)
		assert.NotNil(preregistered)
		assert.Equal(preregistered, r.NewHistogram("summary", 34))

		adHoc = r.NewHistogram("new_summary", 93)
		assert.NotNil(adHoc)
		assert.NotEqual(preregistered, adHoc)
		assert.Equal(adHoc, r.NewHistogram("new_summary", -123))
	})
}

func testRegistryEmptyMetricName(t *testing.T) {
	var (
		assert = assert.New(t)
		r, err = NewRegistry(&Options{
			Metrics: map[string]Metric{
				"": Metric{
					Type: "counter",
				},
			},
		})
	)

	assert.Nil(r)
	assert.Error(err)
}

func testRegistryInvalidType(t *testing.T) {
	var (
		assert = assert.New(t)
		r, err = NewRegistry(&Options{
			Metrics: map[string]Metric{
				"bad": Metric{
					Type: "huh?",
				},
			},
		})
	)

	assert.Nil(r)
	assert.Error(err)
}

func TestRegistry(t *testing.T) {
	t.Run("AsGoKitProvider", testRegistryAsGoKitProvider)
	t.Run("EmptyMetricName", testRegistryEmptyMetricName)
	t.Run("InvalidType", testRegistryInvalidType)
}

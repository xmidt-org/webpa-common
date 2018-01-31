package xmetricstest

import (
	"testing"

	"github.com/Comcast/webpa-common/xmetrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testNewProviderDefault(t *testing.T) {
	assert := assert.New(t)
	assert.NotNil(NewProvider(nil))
}

func testNewProviderGoodConfiguration(t *testing.T) {
	assert := assert.New(t)
	assert.NotNil(NewProvider(
		&xmetrics.Options{
			Metrics: []xmetrics.Metric{
				{Name: "Injected", Type: "counter"},
			},
		},
		func() []xmetrics.Metric {
			return []xmetrics.Metric{
				{Name: "FromModule", Type: "gauge"},
			}
		},
	))
}

func testNewProviderBadConfiguration(t *testing.T) {
	assert := assert.New(t)
	assert.Panics(func() {
		NewProvider(nil, func() []xmetrics.Metric {
			return []xmetrics.Metric{
				{Name: "duplicate", Type: "counter"},
				{Name: "duplicate", Type: "counter"},
			}
		})
	})
}

func testNewProviderUnsupportedType(t *testing.T) {
	assert := assert.New(t)
	assert.Panics(func() {
		NewProvider(nil, func() []xmetrics.Metric {
			return []xmetrics.Metric{
				{Name: "unsupported", Type: "asdfasdfasdfasdf"},
			}
		})
	})
}

func TestNewProvider(t *testing.T) {
	t.Run("Default", testNewProviderDefault)
	t.Run("GoodConfiguration", testNewProviderGoodConfiguration)
	t.Run("BadConfiguration", testNewProviderBadConfiguration)
	t.Run("UnsupportedType", testNewProviderUnsupportedType)
}

func testProviderNewCounter(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		provider = NewProvider(nil, func() []xmetrics.Metric {
			return []xmetrics.Metric{
				{Name: "counter", Type: "counter"},
				{Name: "gauge", Type: "gauge"},
			}
		})
	)

	require.NotNil(provider)

	assert.Panics(func() {
		provider.NewCounter("gauge")
	})

	preconfigured := provider.NewCounter("counter")
	assert.NotNil(preconfigured)
	assert.Implements((*xmetrics.Valuer)(nil), preconfigured)
	assert.True(preconfigured == provider.NewCounter("counter"))

	adhoc := provider.NewCounter("adhoc")
	assert.NotNil(adhoc)
	assert.Implements((*xmetrics.Valuer)(nil), adhoc)
	assert.True(adhoc == provider.NewCounter("adhoc"))
	assert.True(preconfigured != adhoc)
}

func TestProvider(t *testing.T) {
	t.Run("NewCounter", testProviderNewCounter)
}

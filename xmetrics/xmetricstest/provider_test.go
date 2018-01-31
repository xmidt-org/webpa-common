package xmetricstest

import (
	"testing"

	"github.com/Comcast/webpa-common/xmetrics"
	"github.com/stretchr/testify/assert"
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

func TestNewProvider(t *testing.T) {
	t.Run("Default", testNewProviderDefault)
	t.Run("GoodConfiguration", testNewProviderGoodConfiguration)
	t.Run("BadConfiguration", testNewProviderBadConfiguration)
}

package xmetrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func testOptionsDefault(o *Options, t *testing.T) {
	var (
		assert = assert.New(t)
	)

	assert.Equal(DefaultNamespace, o.namespace())
	assert.Equal(DefaultSubsystem, o.subsystem())
	assert.False(o.pedantic())
	assert.False(o.disableGoCollector())
	assert.False(o.disableProcessCollector())
	assert.NotNil(o.registry())
	assert.Empty(o.metrics())
}

func testOptionsCustom(t *testing.T) {
	var (
		assert = assert.New(t)
		o      = Options{
			Namespace:               "custom namespace",
			Subsystem:               "custom subsystem",
			Pedantic:                true,
			DisableGoCollector:      true,
			DisableProcessCollector: true,
			Metrics: map[string]Metric{
				"counter": Metric{
					Type: "counter",
				},
			},
		}
	)

	assert.Equal("custom namespace", o.namespace())
	assert.Equal("custom subsystem", o.subsystem())
	assert.True(o.pedantic())
	assert.True(o.disableGoCollector())
	assert.True(o.disableProcessCollector())
	assert.NotNil(o.registry())
	assert.Equal(
		map[string]Metric{
			"counter": Metric{
				Type: "counter",
			},
		},
		o.metrics(),
	)
}

func TestOptions(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		testOptionsDefault(nil, t)
	})

	t.Run("Default", func(t *testing.T) {
		testOptionsDefault(new(Options), t)
	})

	t.Run("Custom", testOptionsCustom)
}

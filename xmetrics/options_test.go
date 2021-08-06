package xmetrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/webpa-common/v2/logging"
)

func testOptionsDefault(o *Options, t *testing.T) {
	var (
		assert = assert.New(t)
	)

	assert.NotNil(o.logger())
	assert.Equal(DefaultNamespace, o.namespace())
	assert.Equal(DefaultSubsystem, o.subsystem())
	assert.False(o.pedantic())
	assert.False(o.disableGoCollector())
	assert.False(o.disableProcessCollector())
	assert.NotNil(o.registry())
	assert.Empty(o.Module())
}

func testOptionsCustom(t *testing.T) {
	var (
		assert = assert.New(t)
		logger = logging.NewTestLogger(nil, t)
		o      = Options{
			Logger:                  logger,
			Namespace:               "custom namespace",
			Subsystem:               "custom subsystem",
			Pedantic:                true,
			DisableGoCollector:      true,
			DisableProcessCollector: true,
			Metrics: []Metric{
				Metric{
					Name: "counter",
					Type: "counter",
				},
			},
		}
	)

	assert.Equal(logger, o.logger())
	assert.Equal("custom namespace", o.namespace())
	assert.Equal("custom subsystem", o.subsystem())
	assert.True(o.pedantic())
	assert.True(o.disableGoCollector())
	assert.True(o.disableProcessCollector())
	assert.NotNil(o.registry())
	assert.Equal(
		[]Metric{
			Metric{
				Name: "counter",
				Type: "counter",
			},
		},
		o.Module(),
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

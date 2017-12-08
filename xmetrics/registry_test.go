package xmetrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRegistryBasic(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		o = &Options{
			Metrics: map[string]Metric{
				"counter": Metric{
					Type: "counter",
					Help: "a test counter",
				},
			},
		}
	)

	r, err := NewRegistry(o)
	require.NoError(err)
	require.NotNil(r)

	assert.NotNil(r.NewCounter("counter"))
}

func TestRegistry(t *testing.T) {
	t.Run("Basic", testRegistryBasic)
}

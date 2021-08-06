package device

import (
	"testing"

	"github.com/go-kit/kit/metrics/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/webpa-common/v2/xmetrics"
)

func TestMetrics(t *testing.T) {
	var (
		require = require.New(t)
	)

	r, err := xmetrics.NewRegistry(nil, Metrics)
	require.NoError(err)
	require.NotNil(r)

	for _, gaugeName := range []string{DeviceCounter} {
		gauge := r.NewGauge(gaugeName)
		gauge.Add(1.0)
		gauge.Add(-1.0)
	}

	for _, counterName := range []string{RequestResponseCounter, PingCounter, PongCounter, ConnectCounter, DisconnectCounter} {
		counter := r.NewCounter(counterName)
		counter.Add(1.0)
	}
}

func TestNewMeasures(t *testing.T) {
	var (
		assert = assert.New(t)
		m      = NewMeasures(provider.NewDiscardProvider())
	)

	assert.NotNil(m.Device)
	assert.NotNil(m.RequestResponse)
	assert.NotNil(m.Ping)
	assert.NotNil(m.Pong)
	assert.NotNil(m.Connect)
	assert.NotNil(m.Disconnect)
}

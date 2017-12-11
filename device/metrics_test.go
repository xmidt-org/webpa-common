package device

import (
	"testing"

	"github.com/Comcast/webpa-common/xmetrics"
	"github.com/go-kit/kit/metrics/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetrics(t *testing.T) {
	var (
		require = require.New(t)
	)

	r, err := xmetrics.NewRegistry(nil, Metrics)
	require.NoError(err)
	require.NotNil(r)

	for _, name := range []string{DeviceCounter, RequestResponseCounter, PingCounter, PongCounter, ConnectCounter, DisconnectCounter} {
		counter := r.NewCounter(name)
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

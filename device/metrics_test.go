package device

import (
	"testing"

	"github.com/Comcast/webpa-common/xmetrics"
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

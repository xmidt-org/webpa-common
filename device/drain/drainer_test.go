package drain

import (
	"testing"

	"github.com/Comcast/webpa-common/device"
	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDrainerDisconnect(t *testing.T, count int) {
	var (
		_       = assert.New(t)
		require = require.New(t)
		logger  = logging.NewTestLogger(nil, t)

		registry  = new(device.MockRegistry)
		connector = new(device.MockConnector)

		d = New(
			WithLogger(logger),
			WithRegistry(registry),
			WithConnector(connector),
		)
	)

	require.NotNil(d)
}

func TestDrainer(t *testing.T) {
}

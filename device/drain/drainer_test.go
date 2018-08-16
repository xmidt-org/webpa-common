package drain

import (
	"fmt"
	"testing"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDrainerDisconnectAll(t *testing.T, count int) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		logger  = logging.NewTestLogger(nil, t)

		manager = generateManager(assert, 1000)

		d = New(
			WithLogger(logger),
			WithRegistry(manager),
			WithConnector(manager),
		)
	)

	close(manager.pauseVisit)
	require.NotNil(d)
	done, err := d.Start(Job{})
	require.NoError(err)
	require.NotNil(done)

	select {
	case <-done:
		// passed
	case <-time.After(time.Minute):
		assert.Fail("Disconnect all failed to complete")
		return
	}

	assert.Empty(manager.devices)
}

func TestDrainer(t *testing.T) {
	t.Run("DisconnectAll", func(t *testing.T) {
		for _, count := range []int{0, 1, 2, 1709} {
			t.Run(fmt.Sprintf("count=%d", count), func(t *testing.T) {
				testDrainerDisconnectAll(t, count)
			})
		}
	})
}

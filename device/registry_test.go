package device

import (
	"strconv"
	"testing"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/xmetrics/xmetricstest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRegistryAdd(t *testing.T) {
	t.Run("Unlimited", func(t *testing.T) {
		var (
			assert  = assert.New(t)
			require = require.New(t)
			logger  = logging.NewTestLogger(nil, t)

			p = xmetricstest.NewProvider(nil, Metrics)
			r = newRegistry(registryOptions{
				Logger:   logger,
				Measures: NewMeasures(p),
			})
		)

		require.NotNil(r)
		p.Assert(t, DeviceCounter)(xmetricstest.Value(0.0))
		p.Assert(t, ConnectCounter)(xmetricstest.Value(0.0))
		p.Assert(t, DisconnectCounter)(xmetricstest.Value(0.0))
		p.Assert(t, DeviceLimitReachedCounter)(xmetricstest.Value(0.0))
		p.Assert(t, DuplicatesCounter)(xmetricstest.Value(0.0))

		for i := 0; i < 10; i++ {
			d := newDevice(deviceOptions{
				ID:     ID(strconv.Itoa(i)),
				Logger: logger,
			})

			require.NoError(r.add(d))
			assert.False(d.Closed())
			p.Assert(t, DeviceCounter)(xmetricstest.Value(float64(i + 1)))
			p.Assert(t, ConnectCounter)(xmetricstest.Value(float64(i + 1)))
			p.Assert(t, DisconnectCounter)(xmetricstest.Value(0.0))
			p.Assert(t, DeviceLimitReachedCounter)(xmetricstest.Value(0.0))
			p.Assert(t, DuplicatesCounter)(xmetricstest.Value(0.0))
		}

		existing, ok := r.get(ID("0"))
		require.NotNil(existing)
		assert.True(ok)

		duplicate := newDevice(deviceOptions{
			ID:     ID("0"),
			Logger: logger,
		})

		assert.False(existing.Closed())
		assert.False(duplicate.Closed())
		r.add(duplicate)
		p.Assert(t, DeviceCounter)(xmetricstest.Value(10.0))
		p.Assert(t, ConnectCounter)(xmetricstest.Value(11.0))
		p.Assert(t, DisconnectCounter)(xmetricstest.Value(1.0))
		p.Assert(t, DeviceLimitReachedCounter)(xmetricstest.Value(0.0))
		p.Assert(t, DuplicatesCounter)(xmetricstest.Value(1.0))

		assert.True(existing.Closed())
		assert.False(duplicate.Closed())
	})

	t.Run("Limited", func(t *testing.T) {
		var (
			assert  = assert.New(t)
			require = require.New(t)
			logger  = logging.NewTestLogger(nil, t)

			p = xmetricstest.NewProvider(nil, Metrics)
			r = newRegistry(registryOptions{
				Logger:   logger,
				Limit:    1,
				Measures: NewMeasures(p),
			})
		)

		require.NotNil(r)
		p.Assert(t, DeviceCounter)(xmetricstest.Value(0.0))
		p.Assert(t, ConnectCounter)(xmetricstest.Value(0.0))
		p.Assert(t, DisconnectCounter)(xmetricstest.Value(0.0))
		p.Assert(t, DeviceLimitReachedCounter)(xmetricstest.Value(0.0))
		p.Assert(t, DuplicatesCounter)(xmetricstest.Value(0.0))

		initial := newDevice(deviceOptions{
			ID:     ID("test"),
			Logger: logger,
		})

		assert.NoError(r.add(initial))
		assert.False(initial.Closed())
		p.Assert(t, DeviceCounter)(xmetricstest.Value(1.0))
		p.Assert(t, ConnectCounter)(xmetricstest.Value(1.0))
		p.Assert(t, DisconnectCounter)(xmetricstest.Value(0.0))
		p.Assert(t, DeviceLimitReachedCounter)(xmetricstest.Value(0.0))
		p.Assert(t, DuplicatesCounter)(xmetricstest.Value(0.0))

		cantAdd := newDevice(deviceOptions{
			ID:     ID("cantAdd"),
			Logger: logger,
		})

		assert.Error(r.add(cantAdd))
		assert.False(initial.Closed())
		assert.True(cantAdd.Closed())
		p.Assert(t, DeviceCounter)(xmetricstest.Value(1.0))
		p.Assert(t, ConnectCounter)(xmetricstest.Value(1.0))
		p.Assert(t, DisconnectCounter)(xmetricstest.Value(1.0))
		p.Assert(t, DeviceLimitReachedCounter)(xmetricstest.Value(1.0))
		p.Assert(t, DuplicatesCounter)(xmetricstest.Value(0.0))

		duplicate := newDevice(deviceOptions{
			ID:     ID("test"),
			Logger: logger,
		})

		assert.NoError(r.add(duplicate))
		assert.True(initial.Closed())
		assert.False(duplicate.Closed())
		p.Assert(t, DeviceCounter)(xmetricstest.Value(1.0))
		p.Assert(t, ConnectCounter)(xmetricstest.Value(2.0))
		p.Assert(t, DisconnectCounter)(xmetricstest.Value(2.0))
		p.Assert(t, DeviceLimitReachedCounter)(xmetricstest.Value(1.0))
		p.Assert(t, DuplicatesCounter)(xmetricstest.Value(1.0))
	})
}

func testRegistryRemove(t *testing.T) {
}

func TestRegistry(t *testing.T) {
	t.Run("Add", testRegistryAdd)
	t.Run("Remove", testRegistryRemove)
}

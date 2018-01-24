package device

import (
	"math"
	"testing"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/metrics/provider"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func TestOptionsDefault(t *testing.T) {
	assert := assert.New(t)

	for _, o := range []*Options{nil, new(Options)} {
		t.Log(o)

		assert.Equal(DefaultDeviceMessageQueueSize, o.deviceMessageQueueSize())
		assert.NotNil(o.upgrader())
		assert.Equal(uint32(DefaultInitialCapacity), o.initialCapacity())
		assert.Equal(uint32(math.MaxUint32), o.maxDevices())
		assert.Equal(DefaultIdlePeriod, o.idlePeriod())
		assert.Equal(DefaultPingPeriod, o.pingPeriod())
		assert.Equal(DefaultAuthDelay, o.authDelay())
		assert.Equal(DefaultWriteTimeout, o.writeTimeout())
		assert.NotNil(o.logger())
		assert.Empty(o.listeners())
		assert.Equal(provider.NewDiscardProvider(), o.metricsProvider())
	}
}

func TestOptions(t *testing.T) {
	var (
		assert                  = assert.New(t)
		expectedLogger          = logging.DefaultLogger()
		expectedMetricsProvider = provider.NewPrometheusProvider("test", "test")

		o = Options{
			Upgrader: websocket.Upgrader{
				HandshakeTimeout: 12377123 * time.Second,
				ReadBufferSize:   DefaultReadBufferSize + 48729,
				WriteBufferSize:  DefaultWriteBufferSize + 926,
				Subprotocols:     []string{"foobar"},
			},
			InitialCapacity:        DefaultInitialCapacity + 4719,
			MaxDevices:             20000,
			DeviceMessageQueueSize: DefaultDeviceMessageQueueSize + 287342,
			IdlePeriod:             DefaultIdlePeriod + 3472*time.Minute,
			PingPeriod:             DefaultPingPeriod + 384*time.Millisecond,
			AuthDelay:              DefaultAuthDelay + 88*time.Millisecond,
			WriteTimeout:           DefaultWriteTimeout + 327193*time.Second,
			Logger:                 expectedLogger,
			Listeners:              []Listener{func(*Event) {}},
			MetricsProvider:        expectedMetricsProvider,
		}
	)

	assert.Equal(o.DeviceMessageQueueSize, o.deviceMessageQueueSize())
	assert.Equal(
		websocket.Upgrader{
			HandshakeTimeout: 12377123 * time.Second,
			ReadBufferSize:   DefaultReadBufferSize + 48729,
			WriteBufferSize:  DefaultWriteBufferSize + 926,
			Subprotocols:     []string{"foobar"},
		},
		*o.upgrader(),
	)

	assert.Equal(o.InitialCapacity, o.initialCapacity())
	assert.Equal(uint32(20000), o.maxDevices())
	assert.Equal(o.IdlePeriod, o.idlePeriod())
	assert.Equal(o.PingPeriod, o.pingPeriod())
	assert.Equal(o.AuthDelay, o.authDelay())
	assert.Equal(o.WriteTimeout, o.writeTimeout())
	assert.Equal(expectedLogger, o.logger())
	assert.Equal(o.Listeners, o.listeners())
	assert.Equal(expectedMetricsProvider, o.metricsProvider())
}

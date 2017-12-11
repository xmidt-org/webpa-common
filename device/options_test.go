package device

import (
	"math"
	"testing"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/metrics/provider"
	"github.com/stretchr/testify/assert"
)

func TestOptionsDefault(t *testing.T) {
	assert := assert.New(t)

	for _, o := range []*Options{nil, new(Options)} {
		t.Log(o)

		assert.Equal(DefaultDeviceMessageQueueSize, o.deviceMessageQueueSize())
		assert.Equal(DefaultHandshakeTimeout, o.handshakeTimeout())
		assert.Equal(DefaultDecoderPoolSize, o.decoderPoolSize())
		assert.Equal(DefaultEncoderPoolSize, o.encoderPoolSize())
		assert.Equal(uint32(DefaultInitialCapacity), o.initialCapacity())
		assert.Equal(uint32(math.MaxUint32), o.maxDevices())
		assert.Equal(DefaultIdlePeriod, o.idlePeriod())
		assert.Equal(DefaultPingPeriod, o.pingPeriod())
		assert.Equal(DefaultAuthDelay, o.authDelay())
		assert.Equal(DefaultWriteTimeout, o.writeTimeout())
		assert.Equal(DefaultReadBufferSize, o.readBufferSize())
		assert.Equal(DefaultWriteBufferSize, o.writeBufferSize())
		assert.Empty(o.subprotocols())
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
			HandshakeTimeout:       DefaultHandshakeTimeout + 12377123*time.Second,
			DecoderPoolSize:        672393,
			EncoderPoolSize:        1034571,
			InitialCapacity:        DefaultInitialCapacity + 4719,
			MaxDevices:             20000,
			ReadBufferSize:         DefaultReadBufferSize + 48729,
			WriteBufferSize:        DefaultWriteBufferSize + 926,
			Subprotocols:           []string{"foobar"},
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
	assert.Equal(o.HandshakeTimeout, o.handshakeTimeout())
	assert.Equal(o.DecoderPoolSize, o.decoderPoolSize())
	assert.Equal(o.EncoderPoolSize, o.encoderPoolSize())
	assert.Equal(o.InitialCapacity, o.initialCapacity())
	assert.Equal(uint32(20000), o.maxDevices())
	assert.Equal(o.IdlePeriod, o.idlePeriod())
	assert.Equal(o.PingPeriod, o.pingPeriod())
	assert.Equal(o.AuthDelay, o.authDelay())
	assert.Equal(o.WriteTimeout, o.writeTimeout())
	assert.Equal(o.ReadBufferSize, o.readBufferSize())
	assert.Equal(o.WriteBufferSize, o.writeBufferSize())
	assert.Equal(o.Subprotocols, o.subprotocols())
	assert.Equal(expectedLogger, o.logger())
	assert.Equal(o.Listeners, o.listeners())
	assert.Equal(expectedMetricsProvider, o.metricsProvider())
}

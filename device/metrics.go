package device

import (
	"github.com/Comcast/webpa-common/xmetrics"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/provider"
)

const (
	DeviceCounter          = "device_count"
	DuplicatesCounter      = "duplicate_count"
	RequestResponseCounter = "request_response_count"
	PingCounter            = "ping_count"
	PongCounter            = "pong_count"
	ConnectCounter         = "connect_count"
	DisconnectCounter      = "disconnect_count"
)

// Metrics is the device module function that adds default device metrics
func Metrics() []xmetrics.Metric {
	return []xmetrics.Metric{
		xmetrics.Metric{
			Name: DeviceCounter,
			Type: "gauge",
		},
		xmetrics.Metric{
			Name: DuplicatesCounter,
			Type: "counter",
		},
		xmetrics.Metric{
			Name: RequestResponseCounter,
			Type: "counter",
		},
		xmetrics.Metric{
			Name: PingCounter,
			Type: "counter",
		},
		xmetrics.Metric{
			Name: PongCounter,
			Type: "counter",
		},
		xmetrics.Metric{
			Name: ConnectCounter,
			Type: "counter",
		},
		xmetrics.Metric{
			Name: DisconnectCounter,
			Type: "counter",
		},
	}
}

// Measures is a convenient struct that holds all the device-related metric objects for runtime consumption.
type Measures struct {
	Device          metrics.Gauge
	Duplicates      metrics.Counter
	RequestResponse metrics.Counter
	Ping            metrics.Counter
	Pong            metrics.Counter
	Connect         metrics.Counter
	Disconnect      metrics.Counter
}

// NewMeasures constructs a Measures given a go-kit metrics Provider
func NewMeasures(p provider.Provider) Measures {
	return Measures{
		Device:          p.NewGauge(DeviceCounter),
		RequestResponse: p.NewCounter(RequestResponseCounter),
		Ping:            p.NewCounter(PingCounter),
		Pong:            p.NewCounter(PongCounter),
		Connect:         p.NewCounter(ConnectCounter),
		Disconnect:      p.NewCounter(DisconnectCounter),
	}
}

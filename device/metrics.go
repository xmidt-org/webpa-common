package device

import (
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/provider"
	"github.com/xmidt-org/webpa-common/xmetrics"
)

const (
	DeviceCounter             = "device_count"
	DuplicatesCounter         = "duplicate_count"
	RequestResponseCounter    = "request_response_count"
	PingCounter               = "ping_count"
	PongCounter               = "pong_count"
	ConnectCounter            = "connect_count"
	DisconnectCounter         = "disconnect_count"
	DeviceLimitReachedCounter = "device_limit_reached_count"
	ModelGauge                = "hardware_model"
)

// Metrics is the device module function that adds default device metrics
func Metrics() []xmetrics.Metric {
	return []xmetrics.Metric{
		{
			Name: DeviceCounter,
			Type: "gauge",
		},
		{
			Name: DuplicatesCounter,
			Type: "counter",
		},
		{
			Name: RequestResponseCounter,
			Type: "counter",
		},
		{
			Name: PingCounter,
			Type: "counter",
		},
		{
			Name: PongCounter,
			Type: "counter",
		},
		{
			Name: ConnectCounter,
			Type: "counter",
		},
		{
			Name: DisconnectCounter,
			Type: "counter",
		},
		{
			Name: DeviceLimitReachedCounter,
			Type: "counter",
		},
		{
			Name:       ModelGauge,
			Type:       "gauge",
			LabelNames: []string{"model", "partnerid", "firmware"},
		},
	}
}

// Measures is a convenient struct that holds all the device-related metric objects for runtime consumption.
type Measures struct {
	Device          xmetrics.Setter
	LimitReached    xmetrics.Incrementer
	Duplicates      xmetrics.Incrementer
	RequestResponse metrics.Counter
	Ping            xmetrics.Incrementer
	Pong            xmetrics.Incrementer
	Connect         xmetrics.Incrementer
	Disconnect      xmetrics.Adder
	Models          metrics.Gauge
}

// NewMeasures constructs a Measures given a go-kit metrics Provider
func NewMeasures(p provider.Provider) Measures {
	return Measures{
		Device:          p.NewGauge(DeviceCounter),
		LimitReached:    xmetrics.NewIncrementer(p.NewCounter(DeviceLimitReachedCounter)),
		RequestResponse: p.NewCounter(RequestResponseCounter),
		Ping:            xmetrics.NewIncrementer(p.NewCounter(PingCounter)),
		Pong:            xmetrics.NewIncrementer(p.NewCounter(PongCounter)),
		Duplicates:      xmetrics.NewIncrementer(p.NewCounter(DuplicatesCounter)),
		Connect:         xmetrics.NewIncrementer(p.NewCounter(ConnectCounter)),
		Disconnect:      p.NewCounter(DisconnectCounter),
		Models:          p.NewGauge(ModelGauge),
	}
}

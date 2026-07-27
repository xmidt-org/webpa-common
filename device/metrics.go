// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package device

import (
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/provider"

	// nolint:staticcheck
	"github.com/xmidt-org/webpa-common/v2/xmetrics"
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
	WRPSourceCheck            = "wrp_source_check"
)

// Metrics is the device module function that adds default device metrics
func Metrics() []xmetrics.Metric {
	return []xmetrics.Metric{
		{
			Name: DeviceCounter,
			// nolint: goconst
			Type: "gauge",
		},
		{
			Name: DuplicatesCounter,
			// nolint: goconst
			Type: "counter",
		},
		{
			Name: RequestResponseCounter,
			// nolint: goconst
			Type: "counter",
		},
		{
			Name: PingCounter,
			// nolint: goconst
			Type: "counter",
		},
		{
			Name: PongCounter,
			// nolint: goconst
			Type: "counter",
		},
		{
			Name: ConnectCounter,
			// nolint: goconst
			Type: "counter",
		},
		{
			Name: DisconnectCounter,
			// nolint: goconst
			Type: "counter",
		},
		{
			Name: DeviceLimitReachedCounter,
			// nolint: goconst
			Type: "counter",
		},
		{
			Name: ModelGauge,
			// nolint: goconst
			Type: "gauge",
			// nolint: goconst
			LabelNames: []string{"model", "partnerid", "firmware", "trust"},
		},
		{
			Name: WRPSourceCheck,
			// nolint: goconst
			Type: "counter",
			// nolint: goconst
			LabelNames: []string{"outcome", "reason"},
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
	WRPSourceCheck  metrics.Counter
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
		WRPSourceCheck:  p.NewCounter(WRPSourceCheck),
	}
}

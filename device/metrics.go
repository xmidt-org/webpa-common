package device

import "github.com/Comcast/webpa-common/xmetrics"

const (
	DeviceCounter          = "device_count"
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

package server

import "github.com/xmidt-org/webpa-common/xmetrics"

const (
	APIRequestsTotal         = "api_requests_total"
	InFlightRequests         = "in_flight_requests"
	ActiveConnections        = "active_connections"
	RejectedConnections      = "rejected_connections"
	RequestDurationSeconds   = "request_duration_seconds"
	RequestSizeBytes         = "request_size_bytes"
	ResponseSizeBytes        = "response_size_bytes"
	TimeWritingHeaderSeconds = "time_writing_header_seconds"
	MaxProcs                 = "maximum_processors"
)

// Metrics is the module function for this package that adds the default request handling metrics.
// This module is exported for code that does not directly use this package to start a server.
// Never pass this module when using the webpa functions to start a server.
func Metrics() []xmetrics.Metric {
	return []xmetrics.Metric{
		xmetrics.Metric{
			Name:       APIRequestsTotal,
			Type:       "counter",
			Help:       "A counter for requests to the handler",
			LabelNames: []string{"code", "method"},
		},
		xmetrics.Metric{
			Name: InFlightRequests,
			Type: "gauge",
			Help: "A gauge of requests currently being served by the handler.",
		},
		xmetrics.Metric{
			Name:       ActiveConnections,
			Type:       "gauge",
			Help:       "The number of active connections associated with a listener",
			LabelNames: []string{"server"},
		},
		xmetrics.Metric{
			Name:       RejectedConnections,
			Type:       "counter",
			Help:       "The total number of connections rejected due to exceeding the limit",
			LabelNames: []string{"server"},
		},
		xmetrics.Metric{
			Name:    RequestDurationSeconds,
			Type:    "histogram",
			Help:    "A histogram of latencies for requests.",
			Buckets: []float64{0.0625, 0.125, .25, .5, 1, 5, 10, 20, 40, 80, 160},
		},
		xmetrics.Metric{
			Name:    RequestSizeBytes,
			Type:    "histogram",
			Help:    "A histogram of request sizes for requests.",
			Buckets: []float64{200, 500, 900, 1500},
		},
		xmetrics.Metric{
			Name:    ResponseSizeBytes,
			Type:    "histogram",
			Help:    "A histogram of response sizes for requests.",
			Buckets: []float64{200, 500, 900, 1500},
		},
		xmetrics.Metric{
			Name:    TimeWritingHeaderSeconds,
			Type:    "histogram",
			Help:    "A histogram of latencies for writing HTTP headers.",
			Buckets: []float64{0, 1, 2, 3},
		},
		xmetrics.Metric{
			Name: MaxProcs,
			Type: "gauge",
			Help: "The number of current maximum processors this processes is allowed to use.",
		},
	}
}

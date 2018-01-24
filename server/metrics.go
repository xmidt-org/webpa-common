package server

import "github.com/Comcast/webpa-common/xmetrics"

// Metrics is the module function for this package that adds the default request handling metrics.
// This module is exported for code that does not directly use this package to start a server.
// Never pass this module when using the webpa functions to start a server.
func Metrics() []xmetrics.Metric {
	return []xmetrics.Metric{
		xmetrics.Metric{
			Name:       "api_requests_total",
			Type:       "counter",
			Help:       "A counter for requests to the handler",
			LabelNames: []string{"code", "method"},
		},
		xmetrics.Metric{
			Name: "in_flight_requests",
			Type: "gauge",
			Help: "A gauge of requests currently being served by the handler.",
		},
		xmetrics.Metric{
			Name:       "active_connections",
			Type:       "gauge",
			Help:       "The number of active connections associated with a listener",
			LabelNames: []string{"server"},
		},
		xmetrics.Metric{
			Name:    "request_duration_seconds",
			Type:    "histogram",
			Help:    "A histogram of latencies for requests.",
			Buckets: []float64{.25, .5, 1, 2.5, 5, 10},
		},
		xmetrics.Metric{
			Name:    "request_size_bytes",
			Type:    "histogram",
			Help:    "A histogram of request sizes for requests.",
			Buckets: []float64{200, 500, 900, 1500},
		},
		xmetrics.Metric{
			Name:    "response_size_bytes",
			Type:    "histogram",
			Help:    "A histogram of response sizes for requests.",
			Buckets: []float64{200, 500, 900, 1500},
		},
		xmetrics.Metric{
			Name:    "time_writing_header_seconds",
			Type:    "histogram",
			Help:    "A histogram of latencies for writing HTTP headers.",
			Buckets: []float64{0, 1, 2, 3},
		},
	}
}

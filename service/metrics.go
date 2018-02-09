package service

import (
	"github.com/Comcast/webpa-common/xmetrics"
)

const (
	ErrorCount          = "sd_error_count"
	UpdateCount         = "sd_update_count"
	InstanceCount       = "sd_instance_count"
	LastErrorTimestamp  = "sd_last_error_timestamp"
	LastUpdateTimestamp = "sd_last_update_timestamp"
)

// Metrics is the service discovery module function for metrics
func Metrics() []xmetrics.Metric {
	return []xmetrics.Metric{
		{
			Name: ErrorCount,
			Type: "counter",
		},
		{
			Name: UpdateCount,
			Type: "counter",
		},
		{
			Name: InstanceCount,
			Type: "gauge",
		},
		{
			Name: LastErrorTimestamp,
			Type: "gauge",
		},
		{
			Name: LastUpdateTimestamp,
			Type: "gauge",
		},
	}
}

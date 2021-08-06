package service

import (
	"github.com/xmidt-org/webpa-common/v2/xmetrics"
)

const (
	ErrorCount          = "sd_error_count"
	UpdateCount         = "sd_update_count"
	InstanceCount       = "sd_instance_count"
	LastErrorTimestamp  = "sd_last_error_timestamp"
	LastUpdateTimestamp = "sd_last_update_timestamp"

	ServiceLabel  = "service"
	EventKeyLabel = "eventKey"
)

// Metrics is the service discovery module function for metrics
func Metrics() []xmetrics.Metric {
	return []xmetrics.Metric{
		{
			Name:       ErrorCount,
			Type:       "counter",
			Help:       "The total count of errors from the service discovery backend for a particular service",
			LabelNames: []string{ServiceLabel, EventKeyLabel},
		},
		{
			Name:       UpdateCount,
			Type:       "counter",
			Help:       "The total count of updates from the service discovery backend for a particular service",
			LabelNames: []string{ServiceLabel, EventKeyLabel},
		},
		{
			Name:       InstanceCount,
			Type:       "gauge",
			Help:       "The current number of service instances of a given type",
			LabelNames: []string{ServiceLabel, EventKeyLabel},
		},
		{
			Name:       LastErrorTimestamp,
			Type:       "gauge",
			Help:       "The last time the service discovery backend sent an error for a given service",
			LabelNames: []string{ServiceLabel, EventKeyLabel},
		},
		{
			Name:       LastUpdateTimestamp,
			Type:       "gauge",
			Help:       "The last time the service discovery backend sent updated instances for a given service",
			LabelNames: []string{ServiceLabel, EventKeyLabel},
		},
	}
}

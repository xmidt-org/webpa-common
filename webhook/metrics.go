package webhook

import (
	"github.com/Comcast/webpa-common/xmetrics"
	"github.com/go-kit/kit/metrics"
)

const (
	ListSize                     = "webhook_list_size_value"
	NotificationUnmarshallFailed = "notification_unmarshall_failed_count"
)

type WebhookMetrics struct {
	ListSize                     metrics.Gauge
	NotificationUnmarshallFailed metrics.Counter
}

func Metrics() []xmetrics.Metric {
	return []xmetrics.Metric{
		xmetrics.Metric{
			Name: ListSize,
			Help: "Amount of current listeners",
			Type: "gauge",
		},
		xmetrics.Metric{
			Name: NotificationUnmarshallFailed,
			Help: "Count of the number notification messages that failed to unmarshall",
			Type: "counter",
		},
	}
}

func ApplyMetricsData(registry xmetrics.Registry) (m WebhookMetrics) {
	for _, metric := range Metrics() {
		switch metric.Name {
		case ListSize:
			m.ListSize = registry.NewGauge(metric.Name)
			m.ListSize.Add(0.0)
		case NotificationUnmarshallFailed:
			m.NotificationUnmarshallFailed = registry.NewCounter(metric.Name)
			m.NotificationUnmarshallFailed.Add(0.0)
		}
	}
	
	return
}


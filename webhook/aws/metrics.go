package aws

import (
	"github.com/Comcast/webpa-common/xmetrics"
	"github.com/go-kit/kit/metrics"
)

const (
	ListSize                = "webhook_list_size_value"
	SNSNotificationReceived = "webhook_sns_notification_received_count"
	SNSNotificationSent     = "webhook_sns_notification_sent_count"
	SNSSubscribeAttempt     = "webhook_sns_subscribe_attempt_count"
	SNSSubscribed           = "webhook_sns_subscribed_value"
)

type Metrics struct {
	ListSize                metrics.Gauge
	SNSNotificationReceived metrics.Counter
	SNSNotificationSent     metrics.Counter
	SNSSubscribeAttempt     metrics.Counter
	SNSSubscribed           metrics.Gauge
}

func GetMetrics() []xmetrics.Metric {
	return []xmetrics.Metric{
		xmetrics.Metric{
			Name: ListSize,
			Help: "Amount of current listeners",
			Type: "gauge",
		},
		xmetrics.Metric{
			Name:       SNSNotificationReceived,
			Help:       "Count of the number SNS notifications received",
			Type:       "counter",
			LabelNames: []string{"code"},
		},
		xmetrics.Metric{
			Name: SNSNotificationSent,
			Help: "Count of the number SNS notifications received",
			Type: "counter",
		},
		xmetrics.Metric{
			Name:       SNSSubscribeAttempt,
			Help:       "Count of the number of SNS subscription attempts",
			Type:       "counter",
			LabelNames: []string{"code"},
		},
		xmetrics.Metric{
			Name: SNSSubscribed,
			Help: "Is this instance subscribed to SNS",
			Type: "gauge",
		},
	}
}

func AddMetrics(registry xmetrics.Registry) (m Metrics) {
	for _, metric := range GetMetrics() {
		switch metric.Name {
		case ListSize:
			m.ListSize = registry.NewGauge(metric.Name)
			m.ListSize.Add(0.0)
		case SNSNotificationReceived:
			m.SNSNotificationReceived = registry.NewCounter(metric.Name)
			m.SNSNotificationReceived.Add(0.0)
		case SNSNotificationSent:
			m.SNSNotificationSent = registry.NewCounter(metric.Name)
			m.SNSNotificationSent.Add(0.0)
		case SNSSubscribeAttempt:
			m.SNSSubscribeAttempt = registry.NewCounter(metric.Name)
		case SNSSubscribed:
			m.SNSSubscribed = registry.NewGauge(metric.Name)
			m.SNSSubscribed.Add(0.0)
		}
	}
	
	return
}


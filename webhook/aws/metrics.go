package aws

import (
	"github.com/go-kit/kit/metrics"
	"github.com/prometheus/client_golang/prometheus"
	themisXmetrics "github.com/xmidt-org/themis/xmetrics"
	"github.com/xmidt-org/webpa-common/xmetrics"
	"go.uber.org/fx"
)

const (
	DnsReadyQueryCount      = "dns_ready_query_count"
	DnsReady                = "dns_ready"
	SNSNotificationReceived = "webhook_sns_notification_received_count"
	SNSNotificationSent     = "webhook_sns_notification_sent_count"
	SNSSubscribeAttempt     = "webhook_sns_subscribe_attempt_count"
	SNSSubscribed           = "webhook_sns_subscribed_value"
)

type AWSMetrics struct {
	fx.In

	DnsReadyQueryCount      metrics.Counter `name:"dns_ready_query_count"`
	DnsReady                metrics.Gauge   `name:"dns_ready"`
	SNSNotificationReceived metrics.Counter `name:"webhook_sns_notification_received_count"`
	SNSNotificationSent     metrics.Counter `name:"webhook_sns_notification_sent_count"`
	SNSSubscribeAttempt     metrics.Counter `name:"webhook_sns_subscribe_attempt_count"`
	SNSSubscribed           metrics.Gauge   `name:"webhook_sns_subscribed_value"`
}

func ProvideMetrics() fx.Option {
	return fx.Provide(
		themisXmetrics.ProvideCounter(
			prometheus.CounterOpts{
				Name: DnsReadyQueryCount,
				Help: "Count of the number of queries made checking if DNS is ready",
			},
		),
		themisXmetrics.ProvideGauge(
			prometheus.GaugeOpts{
				Name: DnsReady,
				Help: "Is the DNS ready",
			},
		),
		themisXmetrics.ProvideCounter(
			prometheus.CounterOpts{
				Name: SNSNotificationReceived,
				Help: "Count of the number SNS notifications received",
			}, "code",
		),
		themisXmetrics.ProvideCounter(
			prometheus.CounterOpts{
				Name: SNSNotificationSent,
				Help: "Count of the number SNS notifications received",
			},
		),
		themisXmetrics.ProvideCounter(
			prometheus.CounterOpts{
				Name: SNSSubscribeAttempt,
				Help: "Count of the number of SNS subscription attempts",
			}, "code",
		),
		themisXmetrics.ProvideGauge(
			prometheus.GaugeOpts{
				Name: SNSSubscribed,
				Help: "Is this instance subscribed to SNS",
			},
		),
	)
}

// Metrics returns the defined metrics as a list
func Metrics() []xmetrics.Metric {
	return []xmetrics.Metric{
		xmetrics.Metric{
			Name: DnsReadyQueryCount,
			Help: "Count of the number of queries made checking if DNS is ready",
			Type: "counter",
		},
		xmetrics.Metric{
			Name: DnsReady,
			Help: "Is the DNS ready",
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

// ApplyMetricsData is used for setting the counter values on the AWSMetrics
// when stored and accessing for later use
func ApplyMetricsData(registry xmetrics.Registry) (m AWSMetrics) {
	for _, metric := range Metrics() {
		switch metric.Name {
		case DnsReadyQueryCount:
			m.DnsReadyQueryCount = registry.NewCounter(metric.Name)
			m.DnsReadyQueryCount.Add(0.0)
		case DnsReady:
			m.DnsReady = registry.NewGauge(metric.Name)
			m.DnsReady.Add(0.0)
		case SNSNotificationReceived:
			m.SNSNotificationReceived = registry.NewCounter(metric.Name)
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

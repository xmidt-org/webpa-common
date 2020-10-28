package xwebhook

import (
	"github.com/go-kit/kit/metrics"
	"github.com/xmidt-org/webpa-common/xmetrics"
)

// Names
const (
	PollCounter          = "webhook_polls_total"
	WebhookListSizeGauge = "webhook_list_size_value"
)

// Labels
const (
	OutcomeLabel = "outcome"
)

// Label Values
const (
	SuccessOutcome  = "success"
	FailureOutcomme = "failure"
)

// Metrics returns the Metrics relevant to this package
func Metrics() []xmetrics.Metric {
	return []xmetrics.Metric{
		xmetrics.Metric{
			Name:       PollCounter,
			Type:       xmetrics.CounterType,
			Help:       "Counts data polls to fetch webhook items.",
			LabelNames: []string{OutcomeLabel},
		},

		xmetrics.Metric{
			Name: WebhookListSizeGauge,
			Type: xmetrics.GaugeType,
			Help: "Size of the current list of webhooks.",
		},
	}
}

type measures struct {
	pollCount       metrics.Counter
	webhookListSize metrics.Gauge
}

package xwebhook

import (
	"github.com/go-kit/kit/metrics"
	"github.com/xmidt-org/argus/chrysom"
	"github.com/xmidt-org/webpa-common/xmetrics"
)

// Names
const (
	PollCounter          = "webhook_polls_total"
	WebhookListSizeGauge = "xwebhook_list_size_value"
)

// Labels
const (
	OutcomeLabel = "outcome"
)

// Label Values
const (
	SuccessOutcome = "success"
	FailureOutcome = "failure"
)

// Metrics returns the Metrics relevant to this package
func Metrics() []xmetrics.Metric {
	metrics := []xmetrics.Metric{
		{
			Name: WebhookListSizeGauge,
			Type: xmetrics.GaugeType,
			Help: "Size of the current list of webhooks.",
		},
	}
	metrics = append(metrics, chrysom.Metrics()...)
	return metrics
}

type measures struct {
	pollCount       metrics.Counter
	webhookListSize metrics.Gauge
}

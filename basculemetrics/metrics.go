package basculemetrics

import (
	"github.com/go-kit/kit/metrics"
	gokitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	themisXmetrics "github.com/xmidt-org/themis/xmetrics"
	"github.com/xmidt-org/webpa-common/xmetrics"
	"go.uber.org/fx"
)

// Names for our metrics
const (
	AuthValidationOutcome = "auth_validation"
	NBFHistogram          = "auth_from_nbf_seconds"
	EXPHistogram          = "auth_from_exp_seconds"
)

// labels
const (
	OutcomeLabel = "outcome"
	ServerLabel = "server"
)

// Metrics returns the Metrics relevant to this package
func Metrics() []xmetrics.Metric {
	return []xmetrics.Metric{
		{
			Name:       AuthValidationOutcome,
			Type:       xmetrics.CounterType,
			Help:       "Counter for success and failure reason results through bascule",
			LabelNames: []string{ServerLabel, OutcomeLabel},
		},
		{
			Name:    NBFHistogram,
			Type:    xmetrics.HistogramType,
			Help:    "Difference (in seconds) between time of JWT validation and nbf (including leeway)",
			Buckets: []float64{-61, -11, -2, -1, 0, 9, 60}, // defines the upper inclusive (<=) bounds
			LabelNames: []string{ServerLabel},
		},
		{
			Name:    EXPHistogram,
			Type:    xmetrics.HistogramType,
			Help:    "Difference (in seconds) between time of JWT validation and exp (including leeway)",
			Buckets: []float64{-61, -11, -2, -1, 0, 9, 60},
			LabelNames: []string{ServerLabel},
		},
	}
}

func ProvideMetrics() fx.Option {
	return fx.Provide(
		themisXmetrics.ProvideCounter(prometheus.CounterOpts{
			Name:        AuthValidationOutcome,
			Help:        "Counter for the capability checker, providing outcome information by client, partner, and endpoint",
			ConstLabels: nil,
		}, OutcomeLabel),
		themisXmetrics.ProvideHistogram(prometheus.HistogramOpts{
			Name:    NBFHistogram,
			Help:    "Difference (in seconds) between time of JWT validation and nbf (including leeway)",
			Buckets: []float64{-61, -11, -2, -1, 0, 9, 60}, // defines the upper inclusive (<=) bounds
		}),
		themisXmetrics.ProvideHistogram(prometheus.HistogramOpts{
			Name:    EXPHistogram,
			Help:    "Difference (in seconds) between time of JWT validation and exp (including leeway)",
			Buckets: []float64{-61, -11, -2, -1, 0, 9, 60},
		}),
	)
}

func ProvideMetricsVec() fx.Option {
	return fx.Provide(
		themisXmetrics.ProvideCounterVec(prometheus.CounterOpts{
			Name:        AuthValidationOutcome,
			Help:        "Counter for the capability checker, providing outcome information by client, partner, and endpoint",
			ConstLabels: nil,
		}, ServerLabel, OutcomeLabel),
		themisXmetrics.ProvideHistogramVec(prometheus.HistogramOpts{
			Name:    NBFHistogram,
			Help:    "Difference (in seconds) between time of JWT validation and nbf (including leeway)",
			Buckets: []float64{-61, -11, -2, -1, 0, 9, 60}, // defines the upper inclusive (<=) bounds
		}, ServerLabel),
		themisXmetrics.ProvideHistogramVec(prometheus.HistogramOpts{
			Name:    EXPHistogram,
			Help:    "Difference (in seconds) between time of JWT validation and exp (including leeway)",
			Buckets: []float64{-61, -11, -2, -1, 0, 9, 60},
		}, ServerLabel),
	)
}

// AuthValidationMeasures describes the defined metrics that will be used by clients
type AuthValidationMeasures struct {
	fx.In

	NBFHistogram      metrics.Histogram `name:"auth_from_nbf_seconds"`
	ExpHistogram      metrics.Histogram `name:"auth_from_exp_seconds"`
	ValidationOutcome metrics.Counter   `name:"auth_validation"`
}

// NewAuthValidationMeasures realizes desired metrics
func NewAuthValidationMeasures(r xmetrics.Registry) *AuthValidationMeasures {
	return &AuthValidationMeasures{
		NBFHistogram:      gokitprometheus.NewHistogram(r.NewHistogramVec(NBFHistogram)),
		ExpHistogram:      gokitprometheus.NewHistogram(r.NewHistogramVec(EXPHistogram)),
		ValidationOutcome: r.NewCounter(AuthValidationOutcome),
	}
}

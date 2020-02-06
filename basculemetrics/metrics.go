package basculemetrics

import (
	"github.com/go-kit/kit/metrics"
	gokitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/xmidt-org/webpa-common/xmetrics"
)

//Names for our metrics
const (
	AuthValidationOutcome = "auth_validation"
	NBFHistogram          = "auth_from_nbf_seconds"
	EXPHistogram          = "auth_from_exp_seconds"
)

// labels
const (
	OutcomeLabel = "outcome"
)

//Metrics returns the Metrics relevant to this package
func Metrics() []xmetrics.Metric {
	return []xmetrics.Metric{
		xmetrics.Metric{
			Name:       AuthValidationOutcome,
			Type:       xmetrics.CounterType,
			Help:       "Counter for success and failure reason results through bascule",
			LabelNames: []string{OutcomeLabel},
		},
		xmetrics.Metric{
			Name:    NBFHistogram,
			Type:    xmetrics.HistogramType,
			Help:    "Difference (in seconds) between time of JWT validation and nbf (including leeway)",
			Buckets: []float64{-61, -11, -2, -1, 0, 9, 60}, // defines the upper inclusive (<=) bounds
		},
		xmetrics.Metric{
			Name:    EXPHistogram,
			Type:    xmetrics.HistogramType,
			Help:    "Difference (in seconds) between time of JWT validation and exp (including leeway)",
			Buckets: []float64{-61, -11, -2, -1, 0, 9, 60},
		},
	}
}

//AuthValidationMeasures describes the defined metrics that will be used by clients
type AuthValidationMeasures struct {
	NBFHistogram      *gokitprometheus.Histogram
	ExpHistogram      *gokitprometheus.Histogram
	ValidationOutcome metrics.Counter
}

//NewAuthValidationMeasures realizes desired metrics
func NewAuthValidationMeasures(r xmetrics.Registry) *AuthValidationMeasures {
	return &AuthValidationMeasures{
		NBFHistogram:      gokitprometheus.NewHistogram(r.NewHistogramVec(NBFHistogram)),
		ExpHistogram:      gokitprometheus.NewHistogram(r.NewHistogramVec(EXPHistogram)),
		ValidationOutcome: r.NewCounter(AuthValidationOutcome),
	}
}

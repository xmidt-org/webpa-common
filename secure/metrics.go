package secure

import (
	"github.com/Comcast/webpa-common/xmetrics"
	"github.com/go-kit/kit/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

//Names for our metrics
const (
	JWTValidationReasonCounter = "jwt_validation_reason"
	NBFHistogram               = "jwt_from_nbf_seconds"
	EXPHistogram               = "jwt_from_exp_seconds"
)

//Metrics returns the Metrics relevant to this package
func Metrics() []xmetrics.Metric {
	return []xmetrics.Metric{
		xmetrics.Metric{
			Name:       JWTValidationReasonCounter,
			Type:       xmetrics.CounterType,
			Help:       "Counter for validation resolutions per reason",
			LabelNames: []string{"reason"},
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

//JWTValidationMeasures describes the defined metrics that will be used by clients
type JWTValidationMeasures struct {
	NBFHistogram     prometheus.ObserverVec
	ExpHistogram     prometheus.ObserverVec
	ValidationReason metrics.Counter
}

//NewJWTValidationMeasures realizes desired metrics
func NewJWTValidationMeasures(r xmetrics.Registry) *JWTValidationMeasures {
	return &JWTValidationMeasures{
		NBFHistogram:     r.NewHistogramVec(NBFHistogram),
		ExpHistogram:     r.NewHistogramVec(EXPHistogram),
		ValidationReason: r.NewCounter(JWTValidationReasonCounter),
	}
}

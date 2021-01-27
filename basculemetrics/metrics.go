package basculemetrics

import (
	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/metrics"
	gokitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/xmidt-org/themis/xlog"
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
	ServerLabel  = "server"
)

// help messages
const (
	authValidationOutcomeHelpMsg = "Counter for success and failure reason results through bascule"
	nbfHelpMsg                   = "Difference (in seconds) between time of JWT validation and nbf (including leeway)"
	expHelpMsg                   = "Difference (in seconds) between time of JWT validation and exp (including leeway)"
)

// Metrics returns the Metrics relevant to this package targeting our older non uber/fx applications.
// To initialize the metrics, use NewAuthValidationMeasures().
func Metrics() []xmetrics.Metric {
	return []xmetrics.Metric{
		{
			Name:       AuthValidationOutcome,
			Type:       xmetrics.CounterType,
			Help:       authValidationOutcomeHelpMsg,
			LabelNames: []string{OutcomeLabel},
		},
		{
			Name:    NBFHistogram,
			Type:    xmetrics.HistogramType,
			Help:    nbfHelpMsg,
			Buckets: []float64{-61, -11, -2, -1, 0, 9, 60}, // defines the upper inclusive (<=) bounds
		},
		{
			Name:    EXPHistogram,
			Type:    xmetrics.HistogramType,
			Help:    expHelpMsg,
			Buckets: []float64{-61, -11, -2, -1, 0, 9, 60},
		},
	}
}

// ProvideMetrics provides the metrics relevant to this package as uber/fx options.
// This is now deprecated in favor of ProvideMetricsVec.
func ProvideMetrics() fx.Option {
	return fx.Provide(
		themisXmetrics.ProvideCounter(prometheus.CounterOpts{
			Name:        AuthValidationOutcome,
			Help:        authValidationOutcomeHelpMsg,
			ConstLabels: nil,
		}, OutcomeLabel),
		themisXmetrics.ProvideHistogram(prometheus.HistogramOpts{
			Name: NBFHistogram,

			Help:    nbfHelpMsg,
			Buckets: []float64{-61, -11, -2, -1, 0, 9, 60}, // defines the upper inclusive (<=) bounds
		}),
		themisXmetrics.ProvideHistogram(prometheus.HistogramOpts{
			Name:    EXPHistogram,
			Help:    expHelpMsg,
			Buckets: []float64{-61, -11, -2, -1, 0, 9, 60},
		}),
	)
}

// ProvideMetricsVec provides the metrics relevant to this package as uber/fx options.
// The provided metrics are prometheus vectors which gives access to more advanced operations such as CurryWith(labels).
func ProvideMetricsVec() fx.Option {
	return fx.Provide(
		themisXmetrics.ProvideCounterVec(prometheus.CounterOpts{
			Name:        AuthValidationOutcome,
			Help:        authValidationOutcomeHelpMsg,
			ConstLabels: nil,
		}, ServerLabel, OutcomeLabel),
		themisXmetrics.ProvideHistogramVec(prometheus.HistogramOpts{
			Name:    NBFHistogram,
			Help:    nbfHelpMsg,
			Buckets: []float64{-61, -11, -2, -1, 0, 9, 60}, // defines the upper inclusive (<=) bounds
		}, ServerLabel),
		themisXmetrics.ProvideHistogramVec(prometheus.HistogramOpts{
			Name:    EXPHistogram,
			Help:    expHelpMsg,
			Buckets: []float64{-61, -11, -2, -1, 0, 9, 60},
		}, ServerLabel),
	)
}

// AuthValidationMeasures describes the defined metrics that will be used by clients
type AuthValidationMeasures struct {
	fx.In

	NBFHistogram      metrics.Histogram
	ExpHistogram      metrics.Histogram
	ValidationOutcome metrics.Counter
}

// NewAuthValidationMeasures realizes desired metrics. It's intended to be used alongside Metrics() for
// our older non uber/fx applications.
func NewAuthValidationMeasures(r xmetrics.Registry) *AuthValidationMeasures {
	return &AuthValidationMeasures{
		NBFHistogram:      gokitprometheus.NewHistogram(r.NewHistogramVec(NBFHistogram)),
		ExpHistogram:      gokitprometheus.NewHistogram(r.NewHistogramVec(EXPHistogram)),
		ValidationOutcome: r.NewCounter(AuthValidationOutcome),
	}
}

// BaseMeasuresIn is an uber/fx parameter with base metrics ready to be curried into child metrics based on
// custom labels.
type BaseMeasuresIn struct {
	fx.In
	Logger log.Logger

	NBFHistogram      *prometheus.HistogramVec `name:"auth_from_nbf_seconds"`
	ExpHistogram      *prometheus.HistogramVec `name:"auth_from_exp_seconds"`
	ValidationOutcome *prometheus.CounterVec   `name:"auth_validation"`
}

// MeasuresFactory facilitates the creation of child metrics based on server labels.
type MeasuresFactory struct {
	ServerName string
}

// NewMeasures builds the metric listener from the provided raw metrics.
func (m MeasuresFactory) NewMeasures(in BaseMeasuresIn) (*AuthValidationMeasures, error) {
	in.Logger.Log(level.Key(), level.DebugValue(), xlog.MessageKey(), "building auth validation measures", ServerLabel, m.ServerName)
	nbfHistogramVec, err := in.NBFHistogram.CurryWith(prometheus.Labels{ServerLabel: m.ServerName})
	if err != nil {
		return nil, err
	}
	expHistogramVec, err := in.ExpHistogram.CurryWith(prometheus.Labels{ServerLabel: m.ServerName})
	if err != nil {
		return nil, err
	}
	validationOutcomeCounterVec, err := in.ValidationOutcome.CurryWith(prometheus.Labels{ServerLabel: m.ServerName})
	if err != nil {
		return nil, err
	}

	return &AuthValidationMeasures{
		NBFHistogram:      gokitprometheus.NewHistogram(nbfHistogramVec.(*prometheus.HistogramVec)),
		ExpHistogram:      gokitprometheus.NewHistogram(expHistogramVec.(*prometheus.HistogramVec)),
		ValidationOutcome: gokitprometheus.NewCounter(validationOutcomeCounterVec),
	}, nil
}

// Annotated provides the measures as an annotated component with the name "[SERVER]_bascule_validation_measures"
func (m MeasuresFactory) Annotated() fx.Annotated {
	return fx.Annotated{
		Name:   fmt.Sprintf("%s_bascule_validation_measures", m.ServerName),
		Target: m.NewMeasures,
	}
}

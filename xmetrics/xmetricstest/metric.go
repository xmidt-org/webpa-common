package xmetricstest

import (
	"errors"
	"fmt"

	"github.com/Comcast/webpa-common/xmetrics"
	"github.com/go-kit/kit/metrics/generic"
)

// NewMetric creates the appropriate go-kit metrics/generic metric from the
// supplied descriptor.  Both summaries and histograms result in *generic.Histogram instances.
// If the returned error is nil, the returned metric will always be one of the metrics/generic types.
//
// Only the metric Name is used.  Namespace and subsystem are not applied by this factory function.
func NewMetric(m xmetrics.Metric) (interface{}, error) {
	if len(m.Name) == 0 {
		return nil, errors.New("A name is required for a metric")
	}

	switch m.Type {
	case xmetrics.CounterType:
		return generic.NewCounter(m.Name), nil

	case xmetrics.GaugeType:
		return generic.NewGauge(m.Name), nil

	case xmetrics.HistogramType:
		fallthrough

	case xmetrics.SummaryType:
		return generic.NewHistogram(m.Name, len(m.Buckets)), nil

	default:
		return nil, fmt.Errorf("Unsupported metric type: %s", m.Type)
	}
}

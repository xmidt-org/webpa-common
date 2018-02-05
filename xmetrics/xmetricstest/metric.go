package xmetricstest

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/Comcast/webpa-common/xmetrics"
	"github.com/go-kit/kit/metrics/generic"
)

// labelsAndValuesKey produces a consistent, unique key for a set of label/value pairs
func labelsAndValuesKey(labelsAndValues []string) string {
	var (
		count  = len(labelsAndValues)
		output bytes.Buffer
	)

	switch count {
	case 0:
		break

	case 2:

	default:
		if count%2 != 0 {
			panic(errors.New("Each label must be followed by a value"))
		}
	}

	return output.String()
}

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

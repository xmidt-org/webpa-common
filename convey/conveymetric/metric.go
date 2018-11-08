package conveymetric

import (
	"github.com/Comcast/webpa-common/convey"
	"github.com/go-kit/kit/metrics"
)

// UnknownLabel is a constant for when key/tag can not be found in the C JSON
const UnknownLabel = "unknown"

// MetricClosure will be returned after update of struct, this should be used to update the struct, aka decrement the count
type MetricClosure func()

// CMetric provides
type CMetric interface {
	// Update takes the convey JSON to update internal struct, and return a closure to update the struct again, or an
	// error
	//
	// Note: MetricClosure should only be called once.
	Update(data convey.C) (MetricClosure, error)
}

// NewConveyMetric produces a CMetric where gauge is the internal structure to update, tag is the key in the C JSON
// to update the gauge, and label is the `key` for the gauge cardinality.
//
// Note: The Gauge must have the label as one of the constant labels, (aka. the name of the gauge)
func NewConveyMetric(gauge metrics.Gauge, tag string, label string) CMetric {
	return &cMetric{
		tag:   tag,
		label: label,
		gauge: gauge,
	}
}

// cMetric is the internal CMetric implementation
type cMetric struct {
	tag   string
	label string
	gauge metrics.Gauge
}

func (m *cMetric) Update(data convey.C) (MetricClosure, error) {
	key := UnknownLabel
	if item, ok := data[m.tag].(string); ok {
		key = item
	}

	m.gauge.With(m.label, key).Add(1.0)
	return func() { m.gauge.With(m.label, key).Add(-1.0) }, nil
}
package xmetricstest

import (
	"github.com/go-kit/kit/metrics"
	"github.com/xmidt-org/webpa-common/xmetrics"
)

// testingT is the expected behavior for a testing object.  *testing.T implements this interface.
type testingT interface {
	Errorf(string, ...interface{})
}

// expectation is a metric expectation.  The metric will implement one of the go-kit metrics interfaces, e.g. Counter.
type expectation func(t testingT, name string, metric interface{}) bool

// Value returns an expectation for a metric to be of a certain value.  The metric in question must implement
// xmetrics.Valuer, which both counter and gauge do.  This assertion does not constrain the type of metric beyond
// simply exposing a value.  Use another expectation to assert that a metric is of a more specific type.
func Value(expected float64) expectation {
	return func(t testingT, n string, m interface{}) bool {
		v, ok := m.(xmetrics.Valuer)
		if !ok {
			t.Errorf("metric %s does not expose a value (i.e. is not a counter or gauge)", n)
			return false
		}

		if actual := v.Value(); actual != expected {
			t.Errorf("metric %s does not have the expected value %f.  actual value is %f", n, expected, actual)
			return false
		}

		return true
	}
}

// Minimum returns an expectation for a metric to be at least a certain value.  The metric in question
// must implement xmetrics.Valuer, as with the Value expectation.
func Minimum(expected float64) expectation {
	return func(t testingT, n string, m interface{}) bool {
		v, ok := m.(xmetrics.Valuer)
		if !ok {
			t.Errorf("metric %s does not expose a value (i.e. is not a counter or gauge)", n)
			return false
		}

		if actual := v.Value(); actual < expected {
			t.Errorf("metric %s is smaller than the expected value %f.  actual value is %f", n, expected, actual)
			return false
		}

		return true
	}
}

// Counter is an expectation that a certain metric is a counter.  It must implement the go-kit metrics.Counter interface.
func Counter(t testingT, n string, m interface{}) bool {
	_, ok := m.(metrics.Counter)
	if !ok {
		t.Errorf("metric %s is not a counter", n)
	}

	return ok
}

// Gauge is an expectation that a certain metric is a gauge.  It must implement the go-kit metrics.Gauge interface.
func Gauge(t testingT, n string, m interface{}) bool {
	_, ok := m.(metrics.Gauge)
	if !ok {
		t.Errorf("metric %s is not a gauge", n)
	}

	return ok
}

// Histogram is an expectation that a certain metric is a histogram.  It must implement the go-kit metrics.Histogram interface.
func Histogram(t testingT, n string, m interface{}) bool {
	_, ok := m.(metrics.Histogram)
	if !ok {
		t.Errorf("metric %s is not a histogram", n)
	}

	return ok
}

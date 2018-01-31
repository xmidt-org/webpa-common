package xmetricstest

import (
	"fmt"
	"sync"

	"github.com/Comcast/webpa-common/xmetrics"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/generic"
	"github.com/go-kit/kit/metrics/provider"
)

// testingT is the expected behavior for a testing object.  *testing.T implements this interface.
type testingT interface {
	Errorf(string, ...interface{})
}

// Provider is a testing implementation of go-kit's provider.Provider.  Additionally, it provides
// assertion and expectation functionality.
type Provider interface {
	provider.Provider

	// AssertValue asserts that a given metric has a value.  The metric must implement the xmetrics.Valuer interface,
	// which is the case for both counters and gauges.
	AssertValue(testingT, string, float64) bool

	// ExpectValue sets an expectation for a metric having a specific value.  This expectation can be checked
	// with AssertExpectations.
	ExpectValue(testingT, string, float64) Provider

	AssertCounter(testingT, string) *generic.Counter
	AssertCounterValue(testingT, string, float64) *generic.Counter
	ExpectCounter(string) Provider
	ExpectCounterValue(string, float64) Provider

	AssertGauge(testingT, string) *generic.Gauge
	AssertGaugeValue(testingT, string, float64) *generic.Gauge
	ExpectGauge(string) Provider
	ExpectGaugeValue(string, float64) Provider

	AssertHistogram(testingT, string) *generic.Histogram
	ExpectHistogram(string) Provider

	// AssertExpectations verifies all expectations.  It returns true if and only if all
	// expectations pass or if there were no expectations set.
	AssertExpectations(testingT) bool
}

// NewProvider returns a testing Provider instance, using a similar merging algorithm
// as used by xmetrics.NewRegistry.  Namespace and subsystem are not used to determine
// metric uniqueness, which is normally fine since an application tends to use one pair
// of (namespace, subsystem) for all its metrics.
//
// The returned object may be used as a go-kit provider.Provider for testing application code.
// Additionally, it may also be used to set expectations and do assertions on the recorded metrics.
// At this time, label values *are not supported*.
//
// If this function is unable to merge configuration into a Provider, it panics.
func NewProvider(o *xmetrics.Options, m ...xmetrics.Module) Provider {
	merger := xmetrics.NewMerger().
		Namer(func(_, _ string, name string) string { return name }).
		AddModules(false, m...).
		AddModules(true, o.Module)

	if merger.Err() != nil {
		panic(merger.Err())
	}

	return &testProvider{
		metrics: make(map[string]interface{}),
	}
}

type testProvider struct {
	lock         sync.Mutex
	metrics      map[string]interface{}
	expectations []func(t testingT) bool
}

func (tp *testProvider) AssertValue(t testingT, name string, expected float64) bool {
	defer tp.lock.Unlock()
	tp.lock.Lock()

	e, ok := tp.metrics[name]
	if !ok {
		t.Errorf("no such metric: %s", name)
		return false
	}

	v, ok := e.(xmetrics.Valuer)
	if !ok {
		t.Errorf("existing metric does not expose a value (i.e. is not a counter or a gauge): %s", name)
		return false
	}

	actual := v.Value()
	if expected != actual {
		t.Errorf("metric %s does not have the expected value %f.  actual value: %f", name, expected, actual)
		return false
	}

	return true
}

func (tp *testProvider) ExpectValue(t testingT, name string, expected float64) Provider {
	defer tp.lock.Unlock()
	tp.lock.Lock()

	tp.expectations = append(tp.expectations, func(t testingT) bool {
		return tp.AssertValue(t, name, expected)
	})

	return tp
}

func (tp *testProvider) NewCounter(name string) metrics.Counter {
	defer tp.lock.Unlock()
	tp.lock.Lock()

	if e, ok := tp.metrics[name]; ok {
		if c, ok := e.(metrics.Counter); ok {
			return c
		}

		panic(fmt.Errorf("existing metric %s is not a counter", name))
	}

	c := generic.NewCounter(name)
	tp.metrics[name] = c
	return c
}

func (tp *testProvider) AssertCounter(t testingT, name string) *generic.Counter {
	defer tp.lock.Unlock()
	tp.lock.Lock()

	e, ok := tp.metrics[name]
	if !ok {
		t.Errorf("expected counter not present: %s", name)
		return nil
	}

	c, ok := e.(*generic.Counter)
	if !ok {
		t.Errorf("metric %s is not a counter", name)
		return nil
	}

	return c
}

func (tp *testProvider) AssertCounterValue(t testingT, name string, expected float64) *generic.Counter {
	c := tp.AssertCounter(t, name)
	if c != nil {
		actual := c.Value()
		t.Errorf("counter %s does not have the expected value %f.  actual value: %f", name, expected, actual)
		return nil
	}

	return c
}

func (tp *testProvider) ExpectCounter(name string) Provider {
	defer tp.lock.Unlock()
	tp.lock.Lock()

	tp.expectations = append(tp.expectations, func(t testingT) bool {
		return tp.AssertCounter(t, name) != nil
	})

	return tp
}

func (tp *testProvider) ExpectCounterValue(name string, expected float64) Provider {
	defer tp.lock.Unlock()
	tp.lock.Lock()

	tp.expectations = append(tp.expectations, func(t testingT) bool {
		return tp.AssertCounterValue(t, name, expected) != nil
	})

	return tp
}

func (tp *testProvider) NewGauge(name string) metrics.Gauge {
	defer tp.lock.Unlock()
	tp.lock.Lock()

	if e, ok := tp.metrics[name]; ok {
		if g, ok := e.(metrics.Gauge); ok {
			return g
		}

		panic(fmt.Errorf("existing metric %s is not a gauge", name))
	}

	g := generic.NewGauge(name)
	tp.metrics[name] = g
	return g
}

func (tp *testProvider) AssertGauge(t testingT, name string) *generic.Gauge {
	defer tp.lock.Unlock()
	tp.lock.Lock()

	e, ok := tp.metrics[name]
	if !ok {
		t.Errorf("expected gauge not present: %s", name)
		return nil
	}

	g, ok := e.(*generic.Gauge)
	if !ok {
		t.Errorf("metric %s is not a gauge", name)
		return nil
	}

	return g
}

func (tp *testProvider) AssertGaugeValue(t testingT, name string, expected float64) *generic.Gauge {
	g := tp.AssertGauge(t, name)
	if g != nil {
		actual := g.Value()
		t.Errorf("gauge %s does not have the expected value %f.  actual value: %f", name, expected, actual)
		return nil
	}

	return g
}

func (tp *testProvider) ExpectGauge(name string) Provider {
	defer tp.lock.Unlock()
	tp.lock.Lock()

	tp.expectations = append(tp.expectations, func(t testingT) bool {
		return tp.AssertGauge(t, name) != nil
	})

	return tp
}

func (tp *testProvider) ExpectGaugeValue(name string, expected float64) Provider {
	defer tp.lock.Unlock()
	tp.lock.Lock()

	tp.expectations = append(tp.expectations, func(t testingT) bool {
		return tp.AssertGaugeValue(t, name, expected) != nil
	})

	return tp
}

func (tp *testProvider) NewHistogram(name string, buckets int) metrics.Histogram {
	defer tp.lock.Unlock()
	tp.lock.Lock()

	if e, ok := tp.metrics[name]; ok {
		if h, ok := e.(metrics.Histogram); ok {
			return h
		}

		panic(fmt.Errorf("existing metric %s is not a histogram", name))
	}

	h := generic.NewHistogram(name, buckets)
	tp.metrics[name] = h
	return h
}

func (tp *testProvider) AssertHistogram(t testingT, name string) *generic.Histogram {
	defer tp.lock.Unlock()
	tp.lock.Lock()

	e, ok := tp.metrics[name]
	if !ok {
		t.Errorf("expected histogram not present: %s", name)
		return nil
	}

	h, ok := e.(*generic.Histogram)
	if !ok {
		t.Errorf("metric %s is not a histogram", name)
		return nil
	}

	return h
}

func (tp *testProvider) ExpectHistogram(name string) Provider {
	defer tp.lock.Unlock()
	tp.lock.Lock()

	tp.expectations = append(tp.expectations, func(t testingT) bool {
		return tp.AssertHistogram(t, name) != nil
	})

	return tp
}

func (tp *testProvider) Stop() {
}

func (tp *testProvider) AssertExpectations(t testingT) bool {
	defer tp.lock.Unlock()
	tp.lock.Lock()

	result := true
	for _, e := range tp.expectations {
		result = result || e(t)
	}

	return result
}

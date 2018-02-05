package xmetricstest

import (
	"fmt"
	"sync"

	"github.com/Comcast/webpa-common/xmetrics"
	"github.com/Comcast/webpa-common/xmetrics/xmetricstest/expect"
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

	// Expect associates zero or more expectations with a specific metric.  This method adds the expectations
	// to any existing expectations for the metric.
	Expect(string, ...expect.E)

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
// If this function is unable to merge configuration into a Provider, it panics.  The Provider will
// be usable if no options or modules are passed.  Passing configuration is only necessary if the
// actual production configuration is being tested.
func NewProvider(o *xmetrics.Options, m ...xmetrics.Module) Provider {
	merger := xmetrics.NewMerger().
		Namer(func(_, _ string, name string) string { return name }).
		AddModules(false, m...).
		AddModules(true, o.Module)

	if merger.Err() != nil {
		panic(merger.Err())
	}

	tp := &testProvider{
		metrics:      make(map[string]interface{}),
		expectations: make(map[string][]expect.E),
	}

	for name, metric := range merger.Merged() {
		e, err := NewMetric(metric)
		if err != nil {
			panic(err)
		}

		tp.metrics[name] = e
	}

	return tp
}

type testProvider struct {
	lock         sync.Mutex
	metrics      map[string]interface{}
	expectations map[string][]expect.E
}

func (tp *testProvider) NewCounter(name string) metrics.Counter {
	defer tp.lock.Unlock()
	tp.lock.Lock()

	if e, ok := tp.metrics[name]; ok {
		if c, ok := e.(metrics.Counter); ok {
			return c
		}

		panic(fmt.Errorf("metric %s is not a counter", name))
	}

	c := generic.NewCounter(name)
	tp.metrics[name] = c
	return c
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

func (tp *testProvider) NewHistogram(name string, buckets int) metrics.Histogram {
	defer tp.lock.Unlock()
	tp.lock.Lock()

	if e, ok := tp.metrics[name]; ok {
		if h, ok := e.(metrics.Histogram); ok {
			return h
		}

		panic(fmt.Errorf("metric %s is not a histogram", name))
	}

	h := generic.NewHistogram(name, buckets)
	tp.metrics[name] = h
	return h
}

func (tp *testProvider) Stop() {
}

func (tp *testProvider) Expect(name string, e ...expect.E) {
	if len(e) == 0 {
		return
	}

	defer tp.lock.Unlock()
	tp.lock.Lock()
	tp.expectations[name] = append(tp.expectations[name], e...)
}

func (tp *testProvider) AssertExpectations(t testingT) bool {
	defer tp.lock.Unlock()
	tp.lock.Lock()

	result := true
	for name, expectations := range tp.expectations {
		m, ok := tp.metrics[name]
		if !ok {
			result = false
			t.Errorf("no such metric with name: %s", name)
			continue
		}

		for _, e := range expectations {
			result = e(t, name, m) && result
		}
	}

	return result
}

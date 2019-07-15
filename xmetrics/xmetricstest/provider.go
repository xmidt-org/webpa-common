package xmetricstest

import (
	"fmt"
	"sync"

	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/provider"
	"github.com/xmidt-org/webpa-common/xmetrics"
)

// Provider is a testing implementation of go-kit's provider.Provider.  Additionally, it provides
// assertion and expectation functionality.
type Provider interface {
	provider.Provider

	// Expect associates an expectation with a metric.  The optional list of labels and values will
	// examine any nested metric instead of the root metric.  This method uses a Fluent Builder style:
	//
	//    provider.Expect("counter")(xmetricstest.Counter, xmetricstest.Value(1.0)).
	//        Expect("not_found", "code", "404")(xmetricstest.Counter)
	//
	// The returned closure is used to attach expectations to the metric identified by the name and
	// optional label/value pairs.
	Expect(string, ...string) func(...expectation) Provider

	// Assert executes an assertion against this provider immediately, without adding it to the
	// set of expectations asserted via AssertExpectations.
	Assert(testingT, string, ...string) func(...expectation) bool

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
		expectations: make(map[string]map[LVKey][]expectation),
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

// testProvider is the internal Provider implementation that extends go-kit's provider
// with expect/assert functionality.
type testProvider struct {
	lock         sync.Mutex
	metrics      map[string]interface{}
	expectations map[string]map[LVKey][]expectation
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

	c := NewCounter(name)
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

	g := NewGauge(name)
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

	h := NewHistogram(name, buckets)
	tp.metrics[name] = h
	return h
}

func (tp *testProvider) Stop() {
}

func (tp *testProvider) Expect(name string, labelsAndValues ...string) func(...expectation) Provider {
	lvKey, err := NewLVKey(labelsAndValues)
	if err != nil {
		panic(err)
	}

	return func(e ...expectation) Provider {
		defer tp.lock.Unlock()
		tp.lock.Lock()

		labels, ok := tp.expectations[name]
		if !ok {
			labels = make(map[LVKey][]expectation, 1)
			tp.expectations[name] = labels
		}

		labels[lvKey] = append(labels[lvKey], e...)
		return tp
	}
}

func (tp *testProvider) Assert(t testingT, name string, labelsAndValues ...string) func(...expectation) bool {
	lvKey, err := NewLVKey(labelsAndValues)
	if err != nil {
		panic(err)
	}

	return func(e ...expectation) bool {
		defer tp.lock.Unlock()
		tp.lock.Lock()

		metric, ok := tp.metrics[name]
		if !ok {
			t.Errorf("metric %s does not exist", name)
			return false
		}

		metric = metric.(Labeled).Get(lvKey)
		result := true
		for _, f := range e {
			result = f(t, name, metric) && result
		}

		return result
	}
}

func (tp *testProvider) AssertExpectations(t testingT) bool {
	defer tp.lock.Unlock()
	tp.lock.Lock()

	result := true
	for name, labels := range tp.expectations {
		root, ok := tp.metrics[name]
		if !ok {
			t.Errorf("metric %s does not exist", name)
			result = false
			continue
		}

		labeled := root.(Labeled)
		for lvKey, expectations := range labels {
			metric := labeled.Get(lvKey)

			for _, e := range expectations {
				result = e(t, name, metric) && result
			}
		}
	}

	return result
}

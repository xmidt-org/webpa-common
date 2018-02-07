package xmetricstest

import (
	"errors"
	"fmt"
	"sync"

	"github.com/Comcast/webpa-common/xmetrics"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/generic"
)

// rootCounter is a metric which is the root of a label tree of counters.
type rootCounter struct {
	*generic.Counter
	lock sync.Mutex
	tree map[LVKey]metrics.Counter
}

func NewCounter(name string) metrics.Counter {
	c := &rootCounter{
		Counter: generic.NewCounter(name),
		tree:    make(map[LVKey]metrics.Counter),
	}

	c.tree[rootKey] = c
	return c
}

func (c *rootCounter) With(labelsAndValues ...string) metrics.Counter {
	key, err := NewLVKey(labelsAndValues)
	if err != nil {
		panic(err)
	}

	defer c.lock.Unlock()
	c.lock.Lock()

	if existing, ok := c.tree[key]; ok {
		return existing
	}

	nested := &nestedCounter{
		Counter: generic.NewCounter(c.Name),
		with:    c.With,
	}

	c.tree[key] = nested
	return nested
}

func (c *rootCounter) Get(key LVKey) (interface{}, bool) {
	c.lock.Lock()
	existing, ok := c.tree[key]
	c.lock.Unlock()

	return existing, ok
}

// nestedCounter is a non-root counter created by With.
type nestedCounter struct {
	*generic.Counter
	with func(...string) metrics.Counter
}

func (c *nestedCounter) With(labelsAndValues ...string) metrics.Counter {
	return c.with(labelsAndValues...)
}

// rootGauge is a metric which is the root of a label tree of gauges.
type rootGauge struct {
	*generic.Gauge
	lock sync.Mutex
	tree map[LVKey]metrics.Gauge
}

func NewGauge(name string) metrics.Gauge {
	g := &rootGauge{
		Gauge: generic.NewGauge(name),
		tree:  make(map[LVKey]metrics.Gauge),
	}

	g.tree[rootKey] = g
	return g
}

func (g *rootGauge) With(labelsAndValues ...string) metrics.Gauge {
	key, err := NewLVKey(labelsAndValues)
	if err != nil {
		panic(err)
	}

	defer g.lock.Unlock()
	g.lock.Lock()

	if existing, ok := g.tree[key]; ok {
		return existing
	}

	nested := &nestedGauge{
		Gauge: generic.NewGauge(g.Name),
		with:  g.With,
	}

	g.tree[key] = nested
	return nested
}

func (g *rootGauge) Get(key LVKey) (interface{}, bool) {
	g.lock.Lock()
	existing, ok := g.tree[key]
	g.lock.Unlock()

	return existing, ok
}

// nestedGauge is a non-root gauge created by With.
type nestedGauge struct {
	*generic.Gauge
	with func(...string) metrics.Gauge
}

func (nc *nestedGauge) With(labelsAndValues ...string) metrics.Gauge {
	return nc.with(labelsAndValues...)
}

// rootHistogram is a metric which is the root of a label tree of histograms.
type rootHistogram struct {
	*generic.Histogram
	Buckets int
	lock    sync.Mutex
	tree    map[LVKey]metrics.Histogram
}

func NewHistogram(name string, buckets int) metrics.Histogram {
	h := &rootHistogram{
		Histogram: generic.NewHistogram(name, buckets),
		Buckets:   buckets,
		tree:      make(map[LVKey]metrics.Histogram),
	}

	h.tree[rootKey] = h
	return h
}

func (h *rootHistogram) With(labelsAndValues ...string) metrics.Histogram {
	key, err := NewLVKey(labelsAndValues)
	if err != nil {
		panic(err)
	}

	defer h.lock.Unlock()
	h.lock.Lock()

	if existing, ok := h.tree[key]; ok {
		return existing
	}

	nested := &nestedHistogram{
		Histogram: generic.NewHistogram(h.Name, h.Buckets),
		with:      h.With,
	}

	h.tree[key] = nested
	return nested
}

func (h *rootHistogram) Get(key LVKey) (interface{}, bool) {
	h.lock.Lock()
	existing, ok := h.tree[key]
	h.lock.Unlock()

	return existing, ok
}

// nestedHistogram is a non-root gauge created by With.
type nestedHistogram struct {
	*generic.Histogram
	with func(...string) metrics.Histogram
}

func (h *nestedHistogram) With(labelsAndValues ...string) metrics.Histogram {
	return h.with(labelsAndValues...)
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
		return NewCounter(m.Name), nil

	case xmetrics.GaugeType:
		return NewGauge(m.Name), nil

	case xmetrics.HistogramType:
		fallthrough

	case xmetrics.SummaryType:
		return NewHistogram(m.Name, len(m.Buckets)), nil

	default:
		return nil, fmt.Errorf("Unsupported metric type: %s", m.Type)
	}
}

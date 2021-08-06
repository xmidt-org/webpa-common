package xmetricstest

import (
	"errors"
	"fmt"
	"sync"

	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/generic"
	"github.com/xmidt-org/webpa-common/v2/xmetrics"
)

// counter is a testing metric which is the root of a label tree of counters.
type counter struct {
	*generic.Counter
	lock sync.Mutex
	tree map[LVKey]metrics.Counter
}

func NewCounter(name string) metrics.Counter {
	c := &counter{
		Counter: generic.NewCounter(name),
		tree:    make(map[LVKey]metrics.Counter),
	}

	c.tree[rootKey] = c
	return c
}

func (c *counter) With(labelsAndValues ...string) metrics.Counter {
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

func (c *counter) Get(key LVKey) interface{} {
	c.lock.Lock()
	metric, ok := c.tree[key]
	if !ok {
		metric = &nestedCounter{
			Counter: generic.NewCounter(c.Name),
			with:    c.With,
		}

		c.tree[key] = metric
	}

	c.lock.Unlock()
	return metric
}

// nestedCounter is a non-root counter created by With.
type nestedCounter struct {
	*generic.Counter
	with func(...string) metrics.Counter
}

func (c *nestedCounter) With(labelsAndValues ...string) metrics.Counter {
	return c.with(labelsAndValues...)
}

// gauge is a testing metric which is the root of a label tree of gauges.
type gauge struct {
	*generic.Gauge
	lock sync.Mutex
	tree map[LVKey]metrics.Gauge
}

func NewGauge(name string) metrics.Gauge {
	g := &gauge{
		Gauge: generic.NewGauge(name),
		tree:  make(map[LVKey]metrics.Gauge),
	}

	g.tree[rootKey] = g
	return g
}

func (g *gauge) With(labelsAndValues ...string) metrics.Gauge {
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

func (g *gauge) Get(key LVKey) interface{} {
	g.lock.Lock()
	metric, ok := g.tree[key]
	if !ok {
		metric = &nestedGauge{
			Gauge: generic.NewGauge(g.Name),
			with:  g.With,
		}

		g.tree[key] = metric
	}

	g.lock.Unlock()
	return metric
}

// nestedGauge is a non-root gauge created by With.
type nestedGauge struct {
	*generic.Gauge
	with func(...string) metrics.Gauge
}

func (nc *nestedGauge) With(labelsAndValues ...string) metrics.Gauge {
	return nc.with(labelsAndValues...)
}

// histogram is a testing metric which is the root of a label tree of histograms.
type histogram struct {
	*generic.Histogram
	Buckets int
	lock    sync.Mutex
	tree    map[LVKey]metrics.Histogram
}

func NewHistogram(name string, buckets int) metrics.Histogram {
	h := &histogram{
		Histogram: generic.NewHistogram(name, buckets),
		Buckets:   buckets,
		tree:      make(map[LVKey]metrics.Histogram),
	}

	h.tree[rootKey] = h
	return h
}

func (h *histogram) With(labelsAndValues ...string) metrics.Histogram {
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

func (h *histogram) Get(key LVKey) interface{} {
	h.lock.Lock()
	metric, ok := h.tree[key]
	if !ok {
		metric = &nestedHistogram{
			Histogram: generic.NewHistogram(h.Name, h.Buckets),
			with:      h.With,
		}

		h.tree[key] = metric
	}

	h.lock.Unlock()
	return metric
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

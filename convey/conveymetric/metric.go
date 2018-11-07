package conveymetric

import (
	"fmt"
	"github.com/Comcast/webpa-common/convey"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/provider"
)

const UnknownLabel = "unknown"

type MetricClosure func()

type CMetric interface {
	Update(data convey.C) (MetricClosure, error)
}

func NewConveyMetric(provider provider.Provider, tag string, name string) CMetric {
	return &cMetric{
		provider: provider,
		tag:      tag,
		name:     name,
		metrics:  make(map[string]metrics.Gauge),
	}
}

type cMetric struct {
	provider provider.Provider
	tag      string
	name     string
	metrics  map[string]metrics.Gauge
}

func (m *cMetric) Update(data convey.C) (MetricClosure, error) {
	var gague metrics.Gauge

	key := UnknownLabel

	if item, ok := data[m.tag].(string); ok {
		key = item
	}

	if val, found := m.metrics[key]; found {
		gague = val
	} else {
		m.metrics[key] = m.provider.NewGauge(fmt.Sprintf("%s_%s_%s", m.name, m.tag, key))
		gague = m.metrics[key]
	}

	gague.Add(float64(1))
	return func() { gague.Add(float64(-1)) }, nil
}

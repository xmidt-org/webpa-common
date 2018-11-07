package conveymetric

import (
	"fmt"
	"github.com/Comcast/webpa-common/convey"
	"github.com/Comcast/webpa-common/xmetrics"
	"github.com/go-kit/kit/metrics"
)

const UnknownLabel = "unknown"

type MetricClosure func()

type CMetric interface {
	Update(data convey.C) (MetricClosure, error)
	//GetMetrics() []prometheus.Collector
}

func NewConveyMetric(registry xmetrics.Registry, tag string, name string) CMetric {
	return &cMetric{
		Registry: registry,
		Tag:      tag,
		Name:     name,
		metrics:  make(map[string]metrics.Gauge),
	}
}

type cMetric struct {
	Registry xmetrics.Registry
	Tag      string
	Name     string
	metrics  map[string]metrics.Gauge
}

func (m *cMetric) Update(data convey.C) (MetricClosure, error) {
	var gague metrics.Gauge

	key := UnknownLabel

	if item, ok := data[m.Tag].(string); ok {
		key = item
	}

	if val, found := m.metrics[key]; found {
		gague = val
	} else {
		m.metrics[key] = m.Registry.NewGauge(fmt.Sprintf("%s_%s_%s", m.Name, m.Tag, key))
		gague = m.metrics[key]
	}

	gague.Add(float64(1))
	return func() { gague.Add(float64(-1)) }, nil
}

//func (m *cMetric) GetMetrics() []prometheus.Collector {
//	metrics := make([]prometheus.Collector, len(m.metrics))
//
//	index := 0
//	for _, v := range m.metrics {
//		metrics[index] = v
//		index++
//	}
//
//	return metrics
//}

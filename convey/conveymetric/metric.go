package conveymetric

import (
	"fmt"
	"github.com/Comcast/webpa-common/convey"
	"github.com/prometheus/client_golang/prometheus"
)

const UnknownLabel = "unknown"

type MetricClosure func()

type CMetric interface {
	Update(data convey.C) (MetricClosure, error)
	GetMetrics() []prometheus.Collector
}

func NewConveyMetric(tag string, name string) CMetric {
	return &cMetric{
		Tag:     tag,
		Name:    name,
		metrics: make(map[string]prometheus.Gauge),
	}
}

type cMetric struct {
	Tag     string
	Name    string
	metrics map[string]prometheus.Gauge
}

func (m *cMetric) Update(data convey.C) (MetricClosure, error) {
	var gague prometheus.Gauge

	key := UnknownLabel

	if item, ok := data[m.Tag].(string); ok {
		key = item
	}

	if val, found := m.metrics[key]; found {
		gague = val
	} else {
		m.metrics[key] = prometheus.NewGauge(prometheus.GaugeOpts{
			Name:      fmt.Sprintf("%s_%s_%s", m.Name, m.Tag, key),
			Namespace: "convey",
			Subsystem: "convey",
			Help:      fmt.Sprintf("Convey Metrics %s for %s and value %s", m.Name, m.Tag, key),
		})
		gague = m.metrics[key]
	}

	gague.Inc()
	return func() { gague.Dec() }, nil
}

func (m *cMetric) GetMetrics() []prometheus.Collector {
	metrics := make([]prometheus.Collector, len(m.metrics))

	index := 0
	for _, v := range m.metrics {
		metrics[index] = v
		index++
	}

	return metrics
}

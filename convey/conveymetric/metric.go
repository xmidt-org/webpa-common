package conveymetric

import (
	"github.com/Comcast/webpa-common/convey"
	"github.com/go-kit/kit/metrics"
	"regexp"
	"strings"
)

const UnknownLabel = "unknown"

type MetricClosure func()

type CMetric interface {
	Update(data convey.C) (MetricClosure, error)
}

var (
	validKeyRegex   = regexp.MustCompile(`^[a-zA-Z_:]+$`)
	replaceRegex    = regexp.MustCompile(`[\W]+`)
	underscoreRegex = regexp.MustCompile(`_+`)
)

func NewConveyMetric(gauge metrics.Gauge, tag string, metricName string) CMetric {
	return &cMetric{
		tag:   tag,
		metricName:  metricName,
		gauge: gauge,
	}
}

type cMetric struct {
	tag   string
	metricName  string
	gauge metrics.Gauge
}

func (m *cMetric) Update(data convey.C) (MetricClosure, error) {
	key := UnknownLabel
	if item, ok := data[m.tag].(string); ok {
		key = getValidKeyName(item)
	}

	m.gauge.With(m.metricName, key).Add(1.0)
	return func() { m.gauge.With(m.metricName, key).Add(-1.0) }, nil
}

func getValidKeyName(str string) string {
	if validKeyRegex.MatchString(str) {
		return str
	}
	str = strings.TrimSpace(str)
	if str == "" {
		return UnknownLabel
	}

	str = replaceRegex.ReplaceAllLiteralString(str, "_")
	str = strings.Trim(str, "_")
	str = underscoreRegex.ReplaceAllLiteralString(str, "_")

	return str
}

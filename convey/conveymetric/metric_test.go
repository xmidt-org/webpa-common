package conveymetric

import (
	"github.com/Comcast/webpa-common/convey"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConveyMetric(t *testing.T) {
	assert := assert.New(t)

	conveyMetric := NewConveyMetric("hw_model", "HardwareModel")
	data := conveyMetric.GetMetrics()
	assert.Empty(data)

	dec, err := conveyMetric.Update(convey.C{"data": "neat", "hw_model": "apple"})
	assert.NoError(err)
	data = conveyMetric.GetMetrics()
	assert.Len(data, 1)

	descChan := make(chan prometheus.Metric, 10)
	go func() {
		data[0].Collect(descChan)
		close(descChan)
	}()

	for v := range descChan {
		dtoMetric := &dto.Metric{}
		err = v.Write(dtoMetric)
		assert.NoError(err)
		assert.Equal(dtoMetric.GetGauge().GetValue(), float64(1))
	}
	// call back to remove convey from metric
	dec()
	data = conveyMetric.GetMetrics()
	assert.Len(data, 1)

	// 0 metric value
	descChan = make(chan prometheus.Metric, 10)
	go func() {
		data[0].Collect(descChan)
		close(descChan)
	}()

	for v := range descChan {
		dtoMetric := &dto.Metric{}
		err = v.Write(dtoMetric)
		assert.NoError(err)
		assert.Equal(dtoMetric.GetGauge().GetValue(), float64(0))
	}
}

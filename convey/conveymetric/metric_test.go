package conveymetric

import (
	"github.com/Comcast/webpa-common/convey"
	"github.com/Comcast/webpa-common/xmetrics"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConveyMetric(t *testing.T) {
	assert := assert.New(t)

	registry, err := xmetrics.NewRegistry(&xmetrics.Options{
		Namespace: "test",
		Subsystem: "basic",
	})
	assert.NoError(err)
	conveyMetric := NewConveyMetric(registry, "hw_model", "HardwareModel")

	data, err := registry.Gather()
	currentLen := len(data)

	dec, err := conveyMetric.Update(convey.C{"data": "neat", "hw_model": "apple"})
	assert.NoError(err)
	data, err = registry.Gather()
	assert.NoError(err)
	assert.True(currentLen < len(data))
	assert.Len(data[len(data)-1].Metric, 1)
	assert.Equal("test_basic_HardwareModel_hw_model_apple", *data[len(data)-1].Name)
	assert.Equal(float64(1), data[len(data)-1].Metric[0].GetGauge().GetValue())

	// remove the update
	dec()

	data, err = registry.Gather()
	assert.NoError(err)
	assert.Len(data[len(data)-1].Metric, 1)
	assert.Equal(float64(0), data[len(data)-1].Metric[0].GetGauge().GetValue())
	currentLen = len(data)

	// try with now `hw_model`
	dec, err = conveyMetric.Update(convey.C{"data": "neat"})
	assert.NoError(err)
	data, err = registry.Gather()
	assert.NoError(err)
	assert.True(currentLen < len(data))
	assert.Len(data[len(data)-1].Metric, 1)
	assert.Equal("test_basic_HardwareModel_hw_model_unknown", *data[len(data)-1].Name)
	assert.Equal(float64(1), data[len(data)-1].Metric[0].GetGauge().GetValue())

	// remove the update
	dec()

	data, err = registry.Gather()
	assert.NoError(err)
	assert.Len(data[len(data)-1].Metric, 1)
	assert.Equal(float64(0), data[len(data)-1].Metric[0].GetGauge().GetValue())
}

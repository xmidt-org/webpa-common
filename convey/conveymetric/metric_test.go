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
	conveyMetric := NewConveyMetric(registry, "hw-model", "HardwareModel")

	//data, err := registry.Gather()
	expectedName := "test_basic_HardwareModel_hardware123abc"
	assert.False(assertConveyMetric(assert, expectedName, registry, float64(-1)), "metric should not be in registry yet")

	dec, err := conveyMetric.Update(convey.C{"data": "neat", "hw-model": "hardware123abc"})

	assert.True(assertConveyMetric(assert, expectedName, registry, float64(1)))

	// remove the update
	dec()

	assert.True(assertConveyMetric(assert, expectedName, registry, float64(0)))

	// try with no `hw_model`
	expectedName = "test_basic_HardwareModel_unknown"
	dec, err = conveyMetric.Update(convey.C{"data": "neat"})
	assert.True(assertConveyMetric(assert, expectedName, registry, float64(1)))

	// remove the update
	dec()
	assert.True(assertConveyMetric(assert, expectedName, registry, float64(0)))

}

func assertConveyMetric(assert *assert.Assertions, name string, registry xmetrics.Registry, expectedValue float64) bool {
	data, err := registry.Gather()
	assert.NoError(err)
	for i := 0; i < len(data); i ++ {
		if *data[i].Name == name {
			// the test
			assert.Equal(expectedValue, data[i].Metric[0].GetGauge().GetValue())
			return true
		}
	}

	return false
}

func TestGetValidKeyName(t *testing.T) {
	assert := assert.New(t)

	assert.Equal("basic", getValidKeyName("basic"))
	assert.Equal(UnknownLabel, getValidKeyName(""))
	assert.Equal(UnknownLabel, getValidKeyName(" "))
	assert.Equal("hw_model", getValidKeyName("hw-model"))
	assert.Equal("hw_model", getValidKeyName("hw model"))
	assert.Equal("hw_model", getValidKeyName("hw	model"))
	assert.Equal("hw_model", getValidKeyName(" hw	model"))
	assert.Equal("hw_model", getValidKeyName(" hw	model@"))
	assert.Equal("hw_model", getValidKeyName("hw@model"))
	assert.Equal("hw_model", getValidKeyName("hw@ model"))

}

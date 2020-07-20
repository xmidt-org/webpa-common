package conveymetric

import (
	"testing"

	"github.com/xmidt-org/webpa-common/convey"
	"github.com/xmidt-org/webpa-common/xmetrics"
	"github.com/xmidt-org/webpa-common/xmetrics/xmetricstest"

	"github.com/stretchr/testify/assert"
)

func TestConveyMetric(t *testing.T) {
	assert := assert.New(t)

	gauge := xmetricstest.NewGauge("hardware")

	conveyMetric := NewConveyMetric(gauge, []TagLabelPair{{"hw-model", "model"}, {"fw-name", "firmware"}}...)

	dec, err := conveyMetric.Update(convey.C{"data": "neat", "hw-model": "hardware123abc", "fw-name": "firmware-xyz"})
	assert.NoError(err)
	assert.Equal(float64(1), gauge.With("model", "hardware123abc", "firmware", "firmware-xyz").(xmetrics.Valuer).Value())
	// remove the update
	dec()

	assert.Equal(float64(0), gauge.With("model", "hardware123abc", "firmware", "firmware-xyz").(xmetrics.Valuer).Value())

	// try with no `hw_model`
	dec, err = conveyMetric.Update(convey.C{"data": "neat", "fw-name": "firmware-abc"})
	t.Logf("%v+", gauge)
	assert.Equal(float64(1), gauge.With("model", UnknownLabelValue, "firmware", "firmware-abc").(xmetrics.Valuer).Value())

	// remove the update
	dec()
	assert.Equal(float64(0), gauge.With("model", UnknownLabelValue, "firmware", "firmware-abc").(xmetrics.Valuer).Value())
}

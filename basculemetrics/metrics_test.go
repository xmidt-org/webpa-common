package basculemetrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/webpa-common/v2/xmetrics"
)

func newTestAuthValidationMeasure() *AuthValidationMeasures {
	return NewAuthValidationMeasures(xmetrics.MustNewRegistry(nil, Metrics))
}

func TestSimpleRun(t *testing.T) {
	assert := assert.New(t)
	assert.NotNil(newTestAuthValidationMeasure())
}

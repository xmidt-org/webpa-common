package basculechecks

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/webpa-common/xmetrics"
)

func newTestAuthCapabilityCheckMeasure() *AuthCapabilityCheckMeasures {
	return NewAuthCapabilityCheckMeasures(xmetrics.MustNewRegistry(nil, Metrics))
}

func TestSimpleRun(t *testing.T) {
	assert := assert.New(t)
	assert.NotNil(newTestAuthCapabilityCheckMeasure())
}

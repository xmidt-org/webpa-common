package basculechecks

import (
	"testing"

	"github.com/Comcast/webpa-common/xmetrics"
	"github.com/stretchr/testify/assert"
)

func newTestJWTValidationMeasure() *JWTValidationMeasures {
	return NewJWTValidationMeasures(xmetrics.MustNewRegistry(nil, Metrics))
}

func TestSimpleRun(t *testing.T) {
	assert := assert.New(t)
	assert.NotNil(newTestJWTValidationMeasure())
}

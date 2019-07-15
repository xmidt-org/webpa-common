package secure

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/webpa-common/xmetrics"
)

func newTestJWTValidationMeasure() *JWTValidationMeasures {
	return NewJWTValidationMeasures(xmetrics.MustNewRegistry(nil, Metrics))
}

func TestSimpleRun(t *testing.T) {
	assert := assert.New(t)
	assert.NotNil(newTestJWTValidationMeasure())
}

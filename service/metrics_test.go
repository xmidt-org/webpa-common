package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/webpa-common/xmetrics"
)

func TestMetrics(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		r, err = xmetrics.NewRegistry(nil, Metrics)
	)

	require.NoError(err)
	require.NotNil(r)

	assert.NotNil(r.NewCounter(ErrorCount))
	assert.NotNil(r.NewCounter(UpdateCount))
	assert.NotNil(r.NewGauge(InstanceCount))
	assert.NotNil(r.NewGauge(LastErrorTimestamp))
	assert.NotNil(r.NewGauge(LastUpdateTimestamp))
}

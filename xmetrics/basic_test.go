package xmetrics

import (
	"testing"

	"github.com/go-kit/kit/metrics/generic"
	"github.com/stretchr/testify/assert"
)

func TestNewIncrementer(t *testing.T) {
	var (
		assert      = assert.New(t)
		counter     = generic.NewCounter("test")
		incrementer = NewIncrementer(counter)
	)

	assert.Zero(counter.Value())
	incrementer.Inc()
	assert.Equal(1.0, counter.Value())
}

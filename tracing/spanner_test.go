package tracing

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpanner(t *testing.T) {
	var (
		require = require.New(t)
		assert  = assert.New(t)

		expectedStart    = time.Now()
		expectedDuration = time.Duration(23458729347)
		expectedError    = errors.New("expected")

		now = func() time.Time {
			return expectedStart
		}

		since = func(actualStart time.Time) time.Duration {
			assert.Equal(expectedStart, actualStart)
			return expectedDuration
		}

		sp = NewSpanner(Now(now), Since(since))
	)

	require.NotNil(sp)

	finisher := sp.Start("test")
	require.NotNil(finisher)

	span := finisher(expectedError)
	require.NotNil(span)
	assert.Equal("test", span.Name())
	assert.Equal(expectedStart, span.Start())
	assert.Equal(expectedDuration, span.Duration())
	assert.Equal(expectedError, span.Error())

	// idempotent
	assert.Equal(span, finisher(errors.New("this should not get set")))
	assert.Equal("test", span.Name())
	assert.Equal(expectedStart, span.Start())
	assert.Equal(expectedDuration, span.Duration())
	assert.Equal(expectedError, span.Error())
}

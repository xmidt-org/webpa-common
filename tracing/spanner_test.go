// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ExampleSpanner() {
	var (
		now      time.Time
		duration = 100 * time.Millisecond

		// production code can ignore now and since
		// we do this just to get consistent output
		spanner = NewSpanner(
			Now(func() time.Time { return now }),
			Since(func(time.Time) time.Duration { return duration }),
		)

		spans     = make(chan Span, 2)
		firstDone = new(sync.WaitGroup)
	)

	firstDone.Add(1)
	go func() {
		defer firstDone.Done()
		finisher := spanner.Start("success")
		// a successful operation happens here ...
		spans <- finisher(nil)
	}()

	go func() {
		defer close(spans)
		firstDone.Wait()
		finisher := spanner.Start("failure")
		// an operation that fails happens here ...
		spans <- finisher(errors.New("this operation failed"))
	}()

	for s := range spans {
		fmt.Println(s.Name(), s.Start(), s.Duration(), s.Error())
	}

	// Output:
	// success 0001-01-01 00:00:00 +0000 UTC 100ms <nil>
	// failure 0001-01-01 00:00:00 +0000 UTC 100ms this operation failed
}

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

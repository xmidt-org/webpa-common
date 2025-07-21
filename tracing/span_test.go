// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testSpanNoError(t *testing.T) {
	var (
		assert = assert.New(t)
		start  = time.Now()

		s = &span{
			name:  "test",
			start: start,
		}
	)

	assert.Equal("test", s.Name())
	assert.Equal(start, s.Start())
	assert.Zero(s.Duration())
	assert.Nil(s.Error())

	assert.True(s.finish(time.Duration(123), nil))
	assert.Equal("test", s.Name())
	assert.Equal(start, s.Start())
	assert.Equal(time.Duration(123), s.Duration())
	assert.Nil(s.Error())

	assert.False(s.finish(time.Duration(456), errors.New("this should not get set")))
	assert.Equal("test", s.Name())
	assert.Equal(start, s.Start())
	assert.Equal(time.Duration(123), s.Duration())
	assert.Nil(s.Error())
}

func testSpanWithError(t *testing.T) {
	var (
		assert        = assert.New(t)
		start         = time.Now()
		expectedError = errors.New("expected")

		s = &span{
			name:  "test",
			start: start,
		}
	)

	assert.Equal("test", s.Name())
	assert.Equal(start, s.Start())
	assert.Zero(s.Duration())
	assert.Nil(s.Error())

	assert.True(s.finish(time.Duration(123), expectedError))
	assert.Equal("test", s.Name())
	assert.Equal(start, s.Start())
	assert.Equal(time.Duration(123), s.Duration())
	assert.Equal(expectedError, s.Error())

	assert.False(s.finish(time.Duration(456), errors.New("this should not get set")))
	assert.Equal("test", s.Name())
	assert.Equal(start, s.Start())
	assert.Equal(time.Duration(123), s.Duration())
	assert.Equal(expectedError, s.Error())
}

func TestSpan(t *testing.T) {
	t.Run("NoError", testSpanNoError)
	t.Run("WithError", testSpanWithError)
}

// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpanError(t *testing.T) {
	var (
		assert        = assert.New(t)
		expectedError = errors.New("expected")
		spanner       = NewSpanner()
		testSpans     = []Span{
			spanner.Start("first")(nil),
			spanner.Start("second")(errors.New("expected error")),
			spanner.Start("third")(nil),
		}

		withSpans = []Span{
			spanner.Start("fourth")(nil),
			spanner.Start("fifth")(errors.New("another expected error")),
		}

		testData = []struct {
			spanError           SpanError
			expectedError       error
			expectedErrorString string
			expectedSpans       []Span
		}{
			{NewSpanError(nil), nil, NoErrorSupplied, nil},
			{NewSpanError(nil, testSpans[0]), nil, NoErrorSupplied, []Span{testSpans[0]}},
			{NewSpanError(nil, testSpans...), nil, NoErrorSupplied, testSpans},

			{NewSpanError(expectedError), expectedError, "expected", nil},
			{NewSpanError(expectedError, testSpans[0]), expectedError, "expected", []Span{testSpans[0]}},
			{NewSpanError(expectedError, testSpans...), expectedError, "expected", testSpans},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)

		assert.Equal(record.expectedError, record.spanError.Err())
		assert.Equal(record.expectedErrorString, record.spanError.Error())
		assert.Equal(record.expectedSpans, record.spanError.Spans())

		assert.True(record.spanError == record.spanError.WithSpans())

		newError := record.spanError.WithSpans(withSpans...).(SpanError)
		assert.True(record.spanError != newError)
		assert.Equal(record.spanError.Err(), newError.Err())
		assert.Equal(withSpans, newError.Spans())
	}
}

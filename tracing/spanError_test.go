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
			spanError     SpanError
			expectedSpans []Span
		}{
			{NewSpanError(expectedError), nil},
			{NewSpanError(expectedError, testSpans[0]), []Span{testSpans[0]}},
			{NewSpanError(expectedError, testSpans...), testSpans},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)

		assert.Equal(expectedError, record.spanError.Err())
		assert.Equal(expectedError.Error(), record.spanError.Error())
		assert.Equal(record.expectedSpans, record.spanError.Spans())

		assert.True(record.spanError == record.spanError.WithSpans())

		newError := record.spanError.WithSpans(withSpans...).(SpanError)
		assert.True(record.spanError != newError)
		assert.Equal(record.spanError.Err(), newError.Err())
		assert.Equal(withSpans, newError.Spans())
	}
}

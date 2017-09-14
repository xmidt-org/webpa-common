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

		testData = []struct {
			spanError     SpanError
			expectedSpans []Span
		}{
			{NewSpanError(expectedError), nil},
			{NewSpanError(expectedError, &span{}), []Span{&span{}}},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)

		assert.Equal(expectedError, record.spanError.Err())
		assert.Equal(expectedError.Error(), record.spanError.Error())
		assert.Equal(record.expectedSpans, record.spanError.Spans())
	}
}

package tracing

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpanError(t *testing.T) {
	var (
		assert   = assert.New(t)
		testData = []struct {
			spanError     SpanError
			expectedError string
		}{
			{nil, ""},
			{SpanError{}, ""},
			{
				SpanError{&span{err: errors.New("error1")}},
				`"error1"`,
			},
			{
				SpanError{&span{err: errors.New("error1")}, &span{}},
				`"error1"`,
			},
			{
				SpanError{&span{err: errors.New("error1")}, &span{}, &span{err: errors.New("another error")}},
				`"error1","another error"`,
			},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)

		assert.Equal(record.expectedError, record.spanError.String())
		assert.Equal(record.expectedError, record.spanError.Error())
	}
}

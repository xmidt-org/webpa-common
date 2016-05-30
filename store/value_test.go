package store

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type testValue struct {
	wasCalled bool
	value     interface{}
	err       error
}

func (t *testValue) Load() (interface{}, error) {
	t.wasCalled = true
	return t.value, t.err
}

func TestNewValueCache(t *testing.T) {
	assert := assert.New(t)

	var testData = []struct {
		source      testValue
		cachePeriod CachePeriod
		expectError bool
	}{
		{
			testValue{
				value: "success",
			},
			CachePeriodForever,
			false,
		},
		{
			testValue{
				err: errors.New("failure"),
			},
			CachePeriodForever,
			true,
		},
		{
			testValue{
				value: "success",
			},
			CachePeriodNever,
			false,
		},
		{
			testValue{
				err: errors.New("failure"),
			},
			CachePeriodNever,
			false,
		},
		{
			testValue{
				value: "success",
			},
			CachePeriod(30 * time.Hour),
			false,
		},
		{
			testValue{
				err: errors.New("failure"),
			},
			CachePeriod(30 * time.Hour),
			false,
		},
	}

	for _, record := range testData {
		t.Logf("%#v", record)
		actual, err := NewValue(&record.source, record.cachePeriod)

		if assert.Equal(record.expectError, err != nil) && err == nil {
			switch actual.(type) {
			case *singleton:
				assert.Equal(CachePeriodForever, record.cachePeriod)
				assert.True(record.source.wasCalled)

			case *testValue:
				assert.Equal(CachePeriodNever, record.cachePeriod)
				assert.False(record.source.wasCalled)

			case *Cache:
				assert.True(record.cachePeriod > 0)
				assert.False(record.source.wasCalled)

			default:
				t.Fatal("Unexpected Value type")
			}

			actualValue, err := actual.Load()
			assert.Equal(record.source.value, actualValue)
			assert.Equal(record.source.err, err)
			assert.True(record.source.wasCalled)
		}
	}
}

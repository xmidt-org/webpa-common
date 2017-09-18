package tracing

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpans(t *testing.T) {
	var (
		assert    = assert.New(t)
		spanner   = NewSpanner()
		testSpans = []Span{
			spanner.Start("first")(nil),
			spanner.Start("second")(errors.New("expected error")),
			spanner.Start("third")(errors.New("another expected error")),
		}

		testData = []struct {
			container     interface{}
			expectedSpans []Span
			expectedOk    bool
		}{
			{nil, nil, false},
			{"this is not a container", nil, false},
			{testSpans[0], []Span{testSpans[0]}, true},
			{testSpans, testSpans, true},
			{&NopMergeable{S: testSpans}, testSpans, true},
			{&NopMergeable{}, nil, true},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)

		actual, ok := Spans(record.container)
		assert.Equal(record.expectedSpans, actual)
		assert.Equal(record.expectedOk, ok)
	}
}

func TestMergeSpans(t *testing.T) {
	var (
		assert    = assert.New(t)
		spanner   = NewSpanner()
		testSpans = []Span{
			spanner.Start("first")(nil),
			spanner.Start("second")(errors.New("expected error")),
			spanner.Start("third")(errors.New("another expected error")),
			spanner.Start("fourth")(nil),
			spanner.Start("fifth")(errors.New("yet another expected error")),
		}

		emptyContainer    = &NopMergeable{}
		nonEmptyContainer = &NopMergeable{
			S: testSpans[3:],
		}

		nonMergeable = "this is not mergeable"

		testData = []struct {
			originalContainer interface{}
			spans             []interface{}
			expectedContainer interface{}
			expectedOk        bool
		}{
			{nil, nil, nil, false},

			{emptyContainer, nil, emptyContainer, false},
			{emptyContainer, []interface{}{"none", "of", "these", "are", "spans"}, emptyContainer, false},
			{emptyContainer, []interface{}{testSpans[0]}, &NopMergeable{S: []Span{testSpans[0]}}, true},
			{emptyContainer, []interface{}{testSpans}, &NopMergeable{S: testSpans}, true},

			{
				emptyContainer,
				[]interface{}{testSpans[0], testSpans[1:3], nonEmptyContainer},
				&NopMergeable{S: append(append([]Span{testSpans[0]}, testSpans[1:3]...), testSpans[3:]...)},
				true,
			},

			{nonEmptyContainer, nil, nonEmptyContainer, false},
			{nonEmptyContainer, []interface{}{"none", "of", "these", "are", "spans"}, nonEmptyContainer, false},
			{nonEmptyContainer, []interface{}{testSpans[0]}, &NopMergeable{S: append(testSpans[3:], testSpans[0])}, true},
			{nonEmptyContainer, []interface{}{testSpans}, &NopMergeable{S: append(testSpans[3:], testSpans...)}, true},
			{nonEmptyContainer, []interface{}{nonEmptyContainer}, &NopMergeable{S: append(testSpans[3:], testSpans[3:]...)}, true},

			{nonMergeable, nil, nonMergeable, false},
			{nonMergeable, []interface{}{"none", "of", "these", "are", "spans"}, nonMergeable, false},
			{nonMergeable, []interface{}{testSpans[0]}, nonMergeable, false},
			{nonMergeable, []interface{}{testSpans}, nonMergeable, false},
			{nonMergeable, []interface{}{nonEmptyContainer}, nonMergeable, false},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)

		actual, ok := MergeSpans(record.originalContainer, record.spans...)
		assert.Equal(record.expectedContainer, actual)
		assert.Equal(record.expectedOk, ok)
	}
}

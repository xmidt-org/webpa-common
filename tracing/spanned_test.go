// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

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
			container     any
			expectedSpans []Span
			expectedOk    bool
		}{
			{nil, nil, false},
			{"this is not a container", nil, false},
			{testSpans[0], []Span{testSpans[0]}, true},
			{testSpans, testSpans, true},
			{NopMergeable(testSpans), testSpans, true},
			{NopMergeable{}, NopMergeable{}, true},
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

		emptyContainer    = NopMergeable{}
		nonEmptyContainer = NopMergeable(testSpans[3:])

		nonMergeable = "this is not mergeable"

		testData = []struct {
			originalContainer any
			spans             []any
			expectedContainer any
			expectedOk        bool
		}{
			{nil, nil, nil, false},

			{emptyContainer, nil, emptyContainer, false},
			{emptyContainer, []any{"none", "of", "these", "are", "spans"}, emptyContainer, false},
			{emptyContainer, []any{testSpans[0]}, NopMergeable{testSpans[0]}, true},
			{emptyContainer, []any{testSpans}, NopMergeable(testSpans), true},

			{
				emptyContainer,
				[]any{testSpans[0], testSpans[1:3], nonEmptyContainer},
				append(
					append(NopMergeable{testSpans[0]}, testSpans[1:3]...), testSpans[3:]...,
				),
				true,
			},

			{nonEmptyContainer, nil, nonEmptyContainer, false},
			{nonEmptyContainer, []any{"none", "of", "these", "are", "spans"}, nonEmptyContainer, false},
			{nonEmptyContainer, []any{testSpans[0]}, append(NopMergeable(testSpans[3:]), testSpans[0]), true},
			{nonEmptyContainer, []any{testSpans}, append(NopMergeable(testSpans[3:]), testSpans...), true},
			{nonEmptyContainer, []any{nonEmptyContainer}, append(NopMergeable(testSpans[3:]), testSpans[3:]...), true},

			{nonMergeable, nil, nonMergeable, false},
			{nonMergeable, []any{"none", "of", "these", "are", "spans"}, nonMergeable, false},
			{nonMergeable, []any{testSpans[0]}, nonMergeable, false},
			{nonMergeable, []any{testSpans}, nonMergeable, false},
			{nonMergeable, []any{nonEmptyContainer}, nonMergeable, false},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)

		actual, ok := MergeSpans(record.originalContainer, record.spans...)
		assert.Equal(record.expectedContainer, actual)
		assert.Equal(record.expectedOk, ok)
	}
}

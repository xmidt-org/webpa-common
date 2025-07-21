// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package converter

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testDurationValid(t *testing.T) {
	testData := []struct {
		value    string
		expected reflect.Value
	}{
		{"1s", reflect.ValueOf(time.Second)},
		{"20m", reflect.ValueOf(20 * time.Minute)},
		{"3h", reflect.ValueOf(3 * time.Hour)},
	}

	for _, record := range testData {
		t.Run(record.value, func(t *testing.T) {
			assert.Equal(t, record.expected.Interface(), Duration(record.value).Interface())
		})
	}
}

func testDurationInvalid(t *testing.T) {
	testData := []struct {
		value string
	}{
		{""},
		{"asdf"},
	}

	for _, record := range testData {
		t.Run(record.value, func(t *testing.T) {
			assert.False(t, Duration(record.value).IsValid())
		})
	}
}

func TestDuration(t *testing.T) {
	t.Run("Valid", testDurationValid)
	t.Run("Invalid", testDurationInvalid)
}

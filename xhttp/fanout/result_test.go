// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package fanout

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testDefaultShouldTerminate(t *testing.T, statusCode int, expected bool) {
	assert := assert.New(t)
	assert.Equal(
		expected,
		DefaultShouldTerminate(Result{StatusCode: statusCode}),
	)
}

func TestDefaultShouldTerminate(t *testing.T) {
	testData := []struct {
		StatusCode int
		Expected   bool
	}{
		{200, true},
		{201, true},
		{202, true},
		{400, false},
		{404, false},
		{500, false},
		{503, false},
		{504, false},
	}

	for _, record := range testData {
		t.Run(fmt.Sprintf("StatusCode=%d", record.StatusCode), func(t *testing.T) {
			testDefaultShouldTerminate(t, record.StatusCode, record.Expected)
		})
	}
}

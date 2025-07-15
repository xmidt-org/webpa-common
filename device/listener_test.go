// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package device

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func testEventString(t *testing.T) {
	var (
		assert     = assert.New(t)
		values     = make(map[string]bool)
		eventTypes = []EventType{
			Connect,
			Disconnect,
			MessageSent,
			MessageReceived,
			MessageFailed,
			TransactionComplete,
			TransactionBroken,
		}
	)

	for _, eventType := range eventTypes {
		value := eventType.String()
		assert.NotEqual(InvalidEventString, value)
		assert.NotContains(values, value)
		values[value] = true
	}

	assert.Equal(InvalidEventString, EventType(255).String())
}

func TestEvent(t *testing.T) {
	t.Run("String", testEventString)
}

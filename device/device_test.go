// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package device

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/sallust"
	"github.com/xmidt-org/wrp-go/v3"
)

func TestDevice(t *testing.T) {
	var (
		assert              = assert.New(t)
		require             = require.New(t)
		expectedConnectedAt = time.Now().UTC()
		expectedUpTime      = 15 * time.Hour

		testData = []struct {
			expectedID        ID
			expectedQueueSize int
		}{
			{
				ID("ID 1"),
				50,
			},
			{
				ID("ID 2"),
				27,
			},
			{
				ID("ID 3"),
				137,
			},
			{
				ID("ID 4"),
				2,
			},
		}
	)

	for _, record := range testData {
		t.Logf("%v", record)

		var (
			ctx, cancel = context.WithCancel(context.Background())
			// nolint: typecheck
			testMessage = new(wrp.Message)
			device      = newDevice(deviceOptions{
				ID:          record.expectedID,
				QueueSize:   record.expectedQueueSize,
				ConnectedAt: expectedConnectedAt,
				Logger:      sallust.Default(),
				Metadata:    new(Metadata),
			})
		)

		require.NotNil(device)
		assert.NotNil(device.Metadata())
		device.statistics = NewStatistics(func() time.Time { return expectedConnectedAt.Add(expectedUpTime) }, expectedConnectedAt)

		assert.Equal(string(record.expectedID), device.String())
		actualConnectedAt := device.Statistics().ConnectedAt()
		assert.Equal(expectedConnectedAt, actualConnectedAt)

		assert.Equal(record.expectedID, device.ID())
		assert.False(device.Closed())

		assert.Equal(record.expectedID, device.ID())
		assert.Equal(actualConnectedAt, device.Statistics().ConnectedAt())
		assert.False(device.Closed())

		data, err := device.MarshalJSON()
		require.NotEmpty(data)
		require.NoError(err)

		assert.JSONEq(
			fmt.Sprintf(
				`{"id": "%s", "pending": 0, "statistics": {"duplications": 0, "bytesSent": 0, "messagesSent": 0, "bytesReceived": 0, "messagesReceived": 0, "connectedAt": "%s", "upTime": "%s"}}`,
				record.expectedID,
				expectedConnectedAt.UTC().Format(time.RFC3339Nano),
				expectedUpTime,
			),
			string(data),
		)

		for repeat := 0; repeat < record.expectedQueueSize; repeat++ {
			go func() {
				request := (&Request{Message: testMessage}).WithContext(ctx)
				device.Send(request)
			}()
		}

		cancel()

		assert.False(device.Closed())
		device.requestClose(CloseReason{Text: "test"})
		assert.True(device.Closed())
		device.requestClose(CloseReason{Text: "test"})
		assert.True(device.Closed())

		response, err := device.Send(&Request{Message: testMessage})
		assert.Nil(response)
		assert.Error(err)
	}
}

func TestDevice_IntermediateContext(t *testing.T) {
	tests := []struct {
		name                        string
		intermediateContext         string
		expectedIntermediateContext string
	}{
		{
			name:                        "empty intermediate context",
			intermediateContext:         "",
			expectedIntermediateContext: "",
		},
		{
			name:                        "non-empty intermediate context",
			intermediateContext:         "some-context-value",
			expectedIntermediateContext: "some-context-value",
		},
		{
			name:                        "intermediate context with special characters",
			intermediateContext:         "context/with/special-chars_123",
			expectedIntermediateContext: "context/with/special-chars_123",
		},
		{
			name:                        "intermediate context with JSON-like content",
			intermediateContext:         `{"key": "value", "nested": {"data": 123}}`,
			expectedIntermediateContext: `{"key": "value", "nested": {"data": 123}}`,
		},
		{
			name:                        "intermediate context with whitespace",
			intermediateContext:         "  spaced context  ",
			expectedIntermediateContext: "  spaced context  ",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			device := newDevice(deviceOptions{
				ID:        ID("test-device"),
				QueueSize: 10,
				Logger:    sallust.Default(),
				Metadata:  new(Metadata),
			})
			require.NotNil(device)

			// Set the intermediateContext field directly since it's an internal field
			device.intermediateContext = tc.intermediateContext

			assert.Equal(tc.expectedIntermediateContext, device.IntermediateContext())
		})
	}
}

func TestDevice_IntermediateContext_Default(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	// Create a device without setting intermediateContext
	device := newDevice(deviceOptions{
		ID:        ID("test-device"),
		QueueSize: 10,
		Logger:    sallust.Default(),
		Metadata:  new(Metadata),
	})
	require.NotNil(device)

	// Default value should be empty string
	assert.Empty(device.IntermediateContext())
	assert.Equal("", device.IntermediateContext())
}

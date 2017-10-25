package device

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/wrp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			testMessage = new(wrp.Message)
			device      = newDevice(
				record.expectedID,
				record.expectedQueueSize,
				expectedConnectedAt,
				logging.NewTestLogger(nil, t),
			)
		)

		require.NotNil(device)
		device.statistics = NewStatistics(func() time.Time { return expectedConnectedAt.Add(expectedUpTime) }, expectedConnectedAt)

		assert.Equal(string(record.expectedID), device.String())
		actualConnectedAt := device.Statistics().ConnectedAt()
		assert.Equal(expectedConnectedAt, actualConnectedAt)

		assert.Equal(record.expectedID, device.ID())
		assert.False(device.Closed())
		if data, err := json.Marshal(device); assert.Nil(err) {
			// just make sure it's valid JSON
			var unmarshaled map[string]interface{}
			assert.NoError(json.Unmarshal(data, &unmarshaled))
		}

		assert.Equal(record.expectedID, device.ID())
		assert.Equal(actualConnectedAt, device.Statistics().ConnectedAt())
		assert.False(device.Closed())

		var actualJSON bytes.Buffer
		data, err := device.MarshalJSON()
		require.NotEmpty(data)
		require.NoError(err)
		assert.JSONEq(
			fmt.Sprintf(
				`{"id": "%s", "closed": false, "bytesReceived": 0, "bytesSent": 0, "messagesSent": 0, "connectedAt": "%s", "upTime": "%s"}`,
				record.expectedID,
				expectedConnectedAt.Format(time.RFC3339),
				expectedUpTime.String(),
			),
			actualJSON.String(),
		)

		for repeat := 0; repeat < record.expectedQueueSize; repeat++ {
			go func() {
				request := (&Request{Message: testMessage}).WithContext(ctx)
				device.Send(request)
			}()
		}

		cancel()

		assert.False(device.Closed())
		device.requestClose()
		assert.True(device.Closed())
		device.requestClose()
		assert.True(device.Closed())

		response, err := device.Send(&Request{Message: testMessage})
		assert.Nil(response)
		assert.Error(err)
	}
}

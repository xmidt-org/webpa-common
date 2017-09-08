package device

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/wrp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDevice(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

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
			ctx, cancel        = context.WithCancel(context.Background())
			testMessage        = new(wrp.Message)
			minimumConnectedAt = time.Now()
			device             = newDevice(
				record.expectedID,
				record.expectedQueueSize,
				logging.NewTestLogger(nil, t),
			)
		)

		require.NotNil(device)

		t.Log("connection timestamp")
		actualConnectedAt := device.Statistics().ConnectedAt()
		assert.True(minimumConnectedAt.Equal(actualConnectedAt) || minimumConnectedAt.Before(actualConnectedAt))

		t.Log("initial state")
		assert.Equal(record.expectedID, device.ID())
		assert.False(device.Closed())
		if data, err := json.Marshal(device); assert.Nil(err) {
			assert.JSONEq(string(data), device.String())
		}

		t.Log("updateKey should hold other state immutable")
		assert.Equal(record.expectedID, device.ID())
		assert.Equal(actualConnectedAt, device.Statistics().ConnectedAt())
		assert.False(device.Closed())
		if data, err := json.Marshal(device); assert.Nil(err) {
			assert.JSONEq(string(data), device.String())
		}

		for repeat := 0; repeat < record.expectedQueueSize; repeat++ {
			go func() {
				request := (&Request{Message: testMessage}).WithContext(ctx)
				device.Send(request)
			}()
		}

		cancel()

		t.Log("requestClose should be idempotent")
		assert.False(device.Closed())
		device.requestClose()
		assert.True(device.Closed())
		device.requestClose()
		assert.True(device.Closed())

		t.Log("closed state")
		assert.Equal(record.expectedID, device.ID())
		assert.Equal(actualConnectedAt, device.Statistics().ConnectedAt())
		if data, err := json.Marshal(device); assert.Nil(err) {
			assert.JSONEq(string(data), device.String())
		}

		t.Log("Send should fail when device is closed")
		response, err := device.Send(&Request{Message: testMessage})
		assert.Nil(response)
		assert.Error(err)
	}
}

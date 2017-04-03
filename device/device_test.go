package device

import (
	"context"
	"encoding/json"
	"github.com/Comcast/webpa-common/wrp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestDevice(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		testData = []struct {
			expectedID        ID
			initialKey        Key
			updatedKey        Key
			expectedConvey    Convey
			expectedQueueSize int
		}{
			{
				ID("ID 1"),
				Key("initial Key 1"),
				Key("updated Key 1"),
				nil,
				50,
			},
			{
				ID("ID 2"),
				Key("initial Key 2"),
				Key("updated Key 2"),
				Convey{"foo": "bar"},
				27,
			},
			{
				ID("ID 3"),
				Key("initial Key 3"),
				Key("updated Key 3"),
				Convey{"count": 12, "nested": map[string]interface{}{"foo": "bar"}},
				137,
			},
			{
				ID("ID 4"),
				Key("initial Key 4"),
				Key("updated Key 4"),
				Convey{"bad convey": map[interface{}]interface{}{"foo": "bar"}},
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
			device             = newDevice(record.expectedID, record.initialKey, record.expectedConvey, record.expectedQueueSize)
		)

		require.NotNil(device)

		t.Log("connection timestamp")
		actualConnectedAt := device.ConnectedAt()
		assert.True(minimumConnectedAt.Equal(actualConnectedAt) || minimumConnectedAt.Before(actualConnectedAt))

		t.Log("initial state")
		assert.Equal(record.expectedID, device.ID())
		assert.Equal(record.initialKey, device.Key())
		assert.Equal(record.expectedConvey, device.Convey())
		assert.False(device.Closed())
		if data, err := json.Marshal(device); assert.Nil(err) {
			assert.JSONEq(string(data), device.String())
		}

		t.Log("updateKey should hold other state immutable")
		device.updateKey(record.updatedKey)
		assert.Equal(record.expectedID, device.ID())
		assert.Equal(record.updatedKey, device.Key())
		assert.Equal(record.expectedConvey, device.Convey())
		assert.Equal(actualConnectedAt, device.ConnectedAt())
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

		t.Log("RequestClose should be idempotent")
		assert.False(device.Closed())
		device.RequestClose()
		assert.True(device.Closed())
		device.RequestClose()
		assert.True(device.Closed())

		t.Log("closed state")
		assert.Equal(record.expectedID, device.ID())
		assert.Equal(record.updatedKey, device.Key())
		assert.Equal(record.expectedConvey, device.Convey())
		assert.Equal(actualConnectedAt, device.ConnectedAt())
		if data, err := json.Marshal(device); assert.Nil(err) {
			assert.JSONEq(string(data), device.String())
		}

		t.Log("Send should fail when device is closed")
		response, err := device.Send(&Request{Message: testMessage})
		assert.Nil(response)
		assert.Error(err)
	}
}

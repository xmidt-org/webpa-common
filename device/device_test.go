package device

import (
	"encoding/json"
	"github.com/Comcast/webpa-common/wrp"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestDevice(t *testing.T) {
	assert := assert.New(t)
	testData := []struct {
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

	testMessage := new(wrp.Message)

	for _, record := range testData {
		t.Logf("%v", record)
		minimumConnectedAt := time.Now()
		device := newDevice(record.expectedID, record.initialKey, record.expectedConvey, record.expectedQueueSize)
		if !assert.NotNil(device) {
			continue
		}

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

		t.Log("queue size should be honored")
		for repeat := 0; repeat < record.expectedQueueSize; repeat++ {
			if !assert.Nil(device.Send(testMessage)) {
				t.FailNow()
			}
		}

		if sendError := device.Send(testMessage); assert.NotNil(sendError) {
			if busyError, ok := sendError.(DeviceError); assert.True(ok) {
				assert.Equal(device.ID(), busyError.ID())
				assert.Equal(device.Key(), busyError.Key())
			}
		}

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
		if sendError := device.Send(testMessage); assert.NotNil(sendError) {
			if closeError, ok := sendError.(DeviceError); assert.True(ok) {
				assert.Equal(device.ID(), closeError.ID())
				assert.Equal(device.Key(), closeError.Key())
			}
		}
	}
}

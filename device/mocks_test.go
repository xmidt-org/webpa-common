package device

import (
	"github.com/Comcast/webpa-common/wrp"
	"github.com/stretchr/testify/mock"
	"time"
)

// mockRandom provides an io.Reader mock for a source of random bytes
type mockRandom struct {
	mock.Mock
}

func (m *mockRandom) Read(b []byte) (int, error) {
	arguments := m.Called(b)
	return arguments.Int(0), arguments.Error(1)
}

// mockDevice mocks the Interface type
type mockDevice struct {
	mock.Mock
}

func (m *mockDevice) ID() ID {
	arguments := m.Called()
	return arguments.Get(0).(ID)
}

func (m *mockDevice) Key() Key {
	arguments := m.Called()
	return arguments.Get(0).(Key)
}

func (m *mockDevice) Convey() Convey {
	arguments := m.Called()
	return arguments.Get(0).(Convey)
}

func (m *mockDevice) ConnectedAt() time.Time {
	arguments := m.Called()
	return arguments.Get(0).(time.Time)
}

func (m *mockDevice) RequestShutdown() {
	m.Called()
}

func (m *mockDevice) Closed() bool {
	arguments := m.Called()
	return arguments.Bool(0)
}

func (m *mockDevice) Send(message *wrp.Message) error {
	arguments := m.Called(message)
	return arguments.Error(0)
}

// mockDeviceListener provides a single mock for all the device listeners
type mockDeviceListener struct {
	mock.Mock
}

func (m *mockDeviceListener) OnMessage(device Interface, message *wrp.Message) {
	m.Called(device, message)
}

func (m *mockDeviceListener) OnConnect(device Interface) {
	m.Called(device)
}

func (m *mockDeviceListener) OnDisconnect(device Interface) {
	m.Called(device)
}

func (m *mockDeviceListener) OnPong(device Interface, data string) {
	m.Called(device, data)
}

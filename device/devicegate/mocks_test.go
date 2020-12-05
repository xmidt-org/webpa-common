package devicegate

import (
	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/webpa-common/device"
)

type mockDeviceGate struct {
	mock.Mock
}

func (m *mockDeviceGate) VisitAll(visit func(string, Set)) {
	m.Called(visit)
}

func (m *mockDeviceGate) GetFilter(key string) (Set, bool) {
	args := m.Called(key)
	set, _ := args.Get(0).(Set)
	return set, args.Bool(1)
}

func (m *mockDeviceGate) SetFilter(key string, values []interface{}) (Set, bool) {
	args := m.Called(key, values)
	set, _ := args.Get(0).(Set)
	return set, args.Bool(1)
}

func (m *mockDeviceGate) DeleteFilter(key string) bool {
	args := m.Called(key)
	return args.Bool(0)
}

func (m *mockDeviceGate) GetAllowedFilters() (Set, bool) {
	args := m.Called()
	set, _ := args.Get(0).(Set)
	return set, args.Bool(1)
}

func (m *mockDeviceGate) AllowConnection(d device.Interface) (bool, device.MatchResult) {
	args := m.Called(d)
	result, _ := args.Get(1).(device.MatchResult)
	return args.Bool(0), result
}

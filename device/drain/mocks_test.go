package drain

import (
	"sync"

	"github.com/Comcast/webpa-common/device"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockDrainer struct {
	mock.Mock
}

func (m *mockDrainer) Start(j Job) error {
	return m.Called(j).Error(0)
}

func (m *mockDrainer) Status() (bool, Job, Progress) {
	arguments := m.Called()
	return arguments.Bool(0), arguments.Get(1).(Job), arguments.Get(2).(Progress)
}

func (m *mockDrainer) Cancel() (<-chan struct{}, error) {
	arguments := m.Called()
	return arguments.Get(0).(<-chan struct{}), arguments.Error(1)
}

// stubVisitAll creates a mocked device registry with a set of devices stubbed out for
// visitation via VisitAll.  Each invocation to the stub returns a different batch of
// mocked devices, which simulates what would happen if the visited devices were disconnected.
func stubVisitAll(count uint64) (*device.MockRegistry, map[device.ID]bool) {
	var (
		mockRegistry = new(device.MockRegistry)
		next         = 0
		devices      = make([]device.Interface, 0, count)
		ids          = make(map[device.ID]bool, count)
	)

	for mac := uint64(0); mac < count; mac++ {
		var (
			id = device.IntToMAC(mac)
			d  = new(device.MockDevice)
		)

		d.On("ID").Return(id)
		devices = append(devices, d)
		ids[id] = true
	}

	mockRegistry.On("VisitAll", mock.MatchedBy(func(func(device.Interface) bool) bool { return true })).
		Run(func(arguments mock.Arguments) {
			visitor := arguments.Get(0).(func(device.Interface) bool)
			for next < len(devices) {
				result := visitor(devices[next])
				next++
				if !result {
					return
				}
			}
		})

	return mockRegistry, ids
}

// stubDisconnect stubs a device connector to track calls to Disconnect.  The returned sync.Map will hold
// the ids (mapped to booleans) passed to Disconnect().  The result parameter will be the returned value for
// all stubbed calls to Disconnect.
func stubDisconnect(result bool) (*device.MockConnector, *sync.Map) {
	var (
		mockConnector = new(device.MockConnector)
		tracker       = new(sync.Map)
	)

	mockConnector.On("Disconnect", mock.MatchedBy(func(device.Interface) bool { return true })).
		Return(result).
		Run(func(arguments mock.Arguments) {
			id := arguments.Get(0).(device.ID)
			tracker.Store(id, true)
		})

	return mockConnector, tracker
}

// assertTrackedIDs asserts that each tracked id is present in the generated map.  Useful when verifying
// that drained devices did actually get disconnected.
func assertTrackedIDs(a *assert.Assertions, generated map[device.ID]bool, tracked *sync.Map) {
	tracked.Range(func(k, v interface{}) bool {
		id := k.(device.ID)
		a.True(generated[id], "Device id %s not present in generated map", id)
		return true
	})
}

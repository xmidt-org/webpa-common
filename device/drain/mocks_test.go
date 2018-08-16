package drain

import (
	"net/http"
	"sync"

	"github.com/Comcast/webpa-common/device"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockDrainer struct {
	mock.Mock
}

func (m *mockDrainer) Start(j Job) (<-chan struct{}, error) {
	arguments := m.Called(j)
	return arguments.Get(0).(<-chan struct{}), arguments.Error(1)
}

func (m *mockDrainer) Status() (bool, Job, Progress) {
	arguments := m.Called()
	return arguments.Bool(0), arguments.Get(1).(Job), arguments.Get(2).(Progress)
}

func (m *mockDrainer) Cancel() (<-chan struct{}, error) {
	arguments := m.Called()
	return arguments.Get(0).(<-chan struct{}), arguments.Error(1)
}

type stubManager struct {
	lock       sync.RWMutex
	assert     *assert.Assertions
	devices    map[device.ID]device.Interface
	pauseVisit chan struct{}
}

var _ device.Connector = (*stubManager)(nil)
var _ device.Registry = (*stubManager)(nil)

func (sm *stubManager) Connect(http.ResponseWriter, *http.Request, http.Header) (device.Interface, error) {
	sm.assert.Fail("Connect is not supported")
	return nil, nil
}

func (sm *stubManager) Disconnect(id device.ID) bool {
	defer sm.lock.Unlock()
	sm.lock.Lock()

	if _, exists := sm.devices[id]; exists {
		delete(sm.devices, id)
		return true
	}

	return false
}

func (sm *stubManager) DisconnectIf(func(device.ID) bool) int {
	sm.assert.Fail("DisconnectIf is not supported")
	return -1
}

func (sm *stubManager) DisconnectAll() int {
	sm.assert.Fail("DisconnectAll is not supported")
	return -1
}

func (sm *stubManager) Len() int {
	return len(sm.devices)
}

func (sm *stubManager) Get(device.ID) (device.Interface, bool) {
	sm.assert.Fail("Get is not supported")
	return nil, false
}

func (sm *stubManager) VisitAll(p func(device.Interface) bool) (count int) {
	<-sm.pauseVisit
	defer sm.lock.Unlock()
	sm.lock.Lock()

	for _, v := range sm.devices {
		count++
		if !p(v) {
			break
		}
	}

	return
}

func generateManager(assert *assert.Assertions, count uint64) *stubManager {
	sm := &stubManager{
		assert:     assert,
		devices:    make(map[device.ID]device.Interface, count),
		pauseVisit: make(chan struct{}),
	}

	for mac := uint64(0); mac < count; mac++ {
		var (
			id = device.IntToMAC(mac)
			d  = new(device.MockDevice)
		)

		d.On("ID").Return(id)
		d.On("String").Return("mockDevice(" + string(id) + ")")
		sm.devices[id] = d
	}

	return sm
}

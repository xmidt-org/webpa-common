package monitor

import "github.com/stretchr/testify/mock"

type mockListener struct {
	mock.Mock
}

func (m *mockListener) MonitorEvent(e Event) {
	m.Called(e)
}

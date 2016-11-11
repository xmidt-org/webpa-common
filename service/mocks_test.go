package service

import (
	"github.com/strava/go.serversets"
	"github.com/stretchr/testify/mock"
)

type mockRegistrar struct {
	mock.Mock
}

func (m *mockRegistrar) RegisterEndpoint(host string, port int, pingFunc func() error) (*serversets.Endpoint, error) {
	arguments := m.Called(host, port, pingFunc)
	first, _ := arguments.Get(0).(*serversets.Endpoint)
	return first, arguments.Error(1)
}

type mockWatcher struct {
	mock.Mock
}

func (m *mockWatcher) Watch() (*serversets.Watch, error) {
	arguments := m.Called()
	first, _ := arguments.Get(0).(*serversets.Watch)
	return first, arguments.Error(1)
}

type mockAccessor struct {
	mock.Mock
}

func (m *mockAccessor) Get(key []byte) (string, error) {
	arguments := m.Called(key)
	return arguments.String(0), arguments.Error(1)
}

type mockAccessorFactory struct {
	mock.Mock
}

func (m *mockAccessorFactory) New(endpoints []string) Accessor {
	arguments := m.Called(endpoints)
	first, _ := arguments.Get(0).(Accessor)
	return first
}

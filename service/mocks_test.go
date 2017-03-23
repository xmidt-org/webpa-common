package service

import (
	"github.com/strava/go.serversets"
	"github.com/stretchr/testify/mock"
)

func nilPingFunc(actual func() error) bool {
	return actual == nil
}

type mockRegistrar struct {
	mock.Mock
}

func (m *mockRegistrar) RegisterEndpoint(host string, port int, pingFunc func() error) (*serversets.Endpoint, error) {
	arguments := m.Called(host, port, pingFunc)
	first, _ := arguments.Get(0).(*serversets.Endpoint)
	return first, arguments.Error(1)
}

func (m *mockRegistrar) Watch() (Watch, error) {
	arguments := m.Called()
	first, _ := arguments.Get(0).(Watch)
	return first, arguments.Error(1)
}

type mockWatch struct {
	mock.Mock
}

func (m *mockWatch) Close() {
	m.Called()
}

func (m *mockWatch) IsClosed() bool {
	arguments := m.Called()
	return arguments.Bool(0)
}

func (m *mockWatch) Endpoints() []string {
	arguments := m.Called()
	first, _ := arguments.Get(0).([]string)
	return first
}

func (m *mockWatch) Event() <-chan struct{} {
	arguments := m.Called()
	return arguments.Get(0).(<-chan struct{})
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

func (m *mockAccessorFactory) New(endpoints []string) (Accessor, []string) {
	arguments := m.Called(endpoints)
	first, _ := arguments.Get(0).(Accessor)
	second, _ := arguments.Get(1).([]string)
	return first, second
}

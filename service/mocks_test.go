package service

import (
	"github.com/go-kit/kit/sd"
	"github.com/go-kit/kit/sd/zk"
	zkclient "github.com/samuel/go-zookeeper/zk"
	"github.com/stretchr/testify/mock"
)

// resetZkClientFactory resets the global singleton factory function
// to its original value.  This function is handy as a defer for tests.
func resetZkClientFactory() {
	zkClientFactory = zk.NewClient
}

type mockClient struct {
	mock.Mock
}

func (m *mockClient) GetEntries(path string) ([]string, <-chan zkclient.Event, error) {
	arguments := m.Called(path)
	return arguments.Get(0).([]string),
		arguments.Get(1).(<-chan zkclient.Event),
		arguments.Error(2)
}

func (m *mockClient) CreateParentNodes(path string) error {
	return m.Called(path).Error(0)
}

func (m *mockClient) Register(s *zk.Service) error {
	return m.Called(s).Error(0)
}

func (m *mockClient) Deregister(s *zk.Service) error {
	return m.Called(s).Error(0)
}

func (m *mockClient) Stop() {
	m.Called()
}

type mockRegistrar struct {
	mock.Mock
}

func (m *mockRegistrar) Register() {
	m.Called()
}

func (m *mockRegistrar) Deregister() {
	m.Called()
}

type mockInstancer struct {
	mock.Mock
}

func (m *mockInstancer) Register(events chan<- sd.Event) {
	m.Called(events)
}

func (m *mockInstancer) Deregister(events chan<- sd.Event) {
	m.Called(events)
}

type mockAccessor struct {
	mock.Mock
}

func (m *mockAccessor) Get(key []byte) (string, error) {
	arguments := m.Called(key)
	return arguments.String(0), arguments.Error(1)
}

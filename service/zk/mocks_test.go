package zk

import (
	gokitzk "github.com/go-kit/kit/sd/zk"
	zkclient "github.com/samuel/go-zookeeper/zk"
	"github.com/stretchr/testify/mock"
)

// resekClientFactory resets the global singleton factory function
// to its original value.  This function is handy as a defer for tests.
func resetClientFactory() {
	clientFactory = gokitzk.NewClient
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

func (m *mockClient) Register(s *gokitzk.Service) error {
	return m.Called(s).Error(0)
}

func (m *mockClient) Deregister(s *gokitzk.Service) error {
	return m.Called(s).Error(0)
}

func (m *mockClient) Stop() {
	m.Called()
}

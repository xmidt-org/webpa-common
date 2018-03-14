package zk

import (
	"github.com/go-kit/kit/log"
	gokitzk "github.com/go-kit/kit/sd/zk"
	zkclient "github.com/samuel/go-zookeeper/zk"
	"github.com/stretchr/testify/mock"
)

// resekClientFactory resets the global singleton factory function
// to its original value.  This function is handy as a defer for tests.
func resetClientFactory() {
	clientFactory = gokitzk.NewClient
}

// prepareMockClientFactory creates a new mockClientFactory and sets up this package
// to use it.
func prepareMockClientFactory() *mockClientFactory {
	m := new(mockClientFactory)
	clientFactory = m.NewClient
	return m
}

type mockClientFactory struct {
	mock.Mock
}

func (m *mockClientFactory) NewClient(servers []string, logger log.Logger, options ...gokitzk.Option) (gokitzk.Client, error) {
	arguments := m.Called(servers, logger, options)

	first, _ := arguments.Get(0).(gokitzk.Client)
	return first, arguments.Error(1)
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

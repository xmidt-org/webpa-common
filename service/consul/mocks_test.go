package consul

import (
	gokitconsul "github.com/go-kit/kit/sd/consul"
	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/mock"
)

// resekClientFactory resets the global singleton factory function
// to its original value.  This function is handy as a defer for tests.
func resetClientFactory() {
	clientFactory = gokitconsul.NewClient
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

func (m *mockClientFactory) NewClient(c *api.Client) gokitconsul.Client {
	return m.Called(c).Get(0).(gokitconsul.Client)
}

type mockClient struct {
	mock.Mock
}

func (m *mockClient) Register(r *api.AgentServiceRegistration) error {
	return m.Called(r).Error(0)
}

func (m *mockClient) Deregister(r *api.AgentServiceRegistration) error {
	return m.Called(r).Error(0)
}

func (m *mockClient) Service(service, tag string, passingOnly bool, queryOpts *api.QueryOptions) ([]*api.ServiceEntry, *api.QueryMeta, error) {
	var (
		arguments = m.Called(service, tag, passingOnly, queryOpts)
		first, _  = arguments.Get(0).([]*api.ServiceEntry)
		second, _ = arguments.Get(1).(*api.QueryMeta)
	)

	return first, second, arguments.Error(2)
}

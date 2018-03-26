package consul

import (
	"time"

	gokitconsul "github.com/go-kit/kit/sd/consul"
	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/mock"
)

// resekClientFactory resets the global singleton factory function
// to its original value.  This function is handy as a defer for tests.
func resetClientFactory() {
	clientFactory = defaultClientFactory
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

func (m *mockClientFactory) NewClient(c *api.Client) (gokitconsul.Client, ttlUpdater) {
	arguments := m.Called(c)
	return arguments.Get(0).(gokitconsul.Client),
		arguments.Get(1).(ttlUpdater)
}

func resetTickerFactory() {
	tickerFactory = defaultTickerFactory
}

func prepareMockTickerFactory() *mockTickerFactory {
	m := new(mockTickerFactory)
	tickerFactory = m.NewTicker
	return m
}

type mockTickerFactory struct {
	mock.Mock
}

func (m *mockTickerFactory) NewTicker(d time.Duration) (<-chan time.Time, func()) {
	arguments := m.Called(d)
	return arguments.Get(0).(<-chan time.Time), arguments.Get(1).(func())
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

type mockTTLUpdater struct {
	mock.Mock
}

func (m *mockTTLUpdater) UpdateTTL(checkID, output, status string) error {
	return m.Called(checkID, output, status).Error(0)
}

package consul

import (
	"testing"

	"github.com/Comcast/webpa-common/logging"
	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func testNewEnvironmentEmpty(t *testing.T) {
	defer resetClientFactory()

	var (
		assert        = assert.New(t)
		clientFactory = prepareMockClientFactory()
	)

	e, err := NewEnvironment(nil, "http", Options{})
	assert.Nil(e)
	assert.NoError(err)

	clientFactory.AssertExpectations(t)
}

func testNewEnvironmentClientError(t *testing.T) {
	defer resetClientFactory()

	var (
		assert        = assert.New(t)
		clientFactory = prepareMockClientFactory()

		co = Options{
			Client: &api.Config{
				Address: "here is an unknown scheme://grabthar.hammer.com:1856",
			},
			Watches: []Watch{
				Watch{
					Service:     "foobar",
					Tags:        []string{"tag1"},
					PassingOnly: true,
				},
			},
		}
	)

	e, err := NewEnvironment(nil, "http", co)
	assert.Nil(e)
	assert.Error(err)

	clientFactory.AssertExpectations(t)
}

func testNewEnvironmentFull(t *testing.T) {
	defer resetClientFactory()

	var (
		assert  = assert.New(t)
		require = require.New(t)

		logger        = logging.NewTestLogger(nil, t)
		clientFactory = prepareMockClientFactory()
		client        = new(mockClient)
		ttlUpdater    = new(mockTTLUpdater)

		co = Options{
			Client: &api.Config{
				Address: "localhost:8500",
				Scheme:  "https",
			},
			Registrations: []api.AgentServiceRegistration{
				api.AgentServiceRegistration{
					ID:      "service1",
					Address: "grubly.com",
					Port:    1111,
				},
				api.AgentServiceRegistration{
					ID:      "service2",
					Address: "grubly.com",
					Port:    1111,
				}, // duplicates should be ignored
			},
			Watches: []Watch{
				Watch{
					Service:     "foobar",
					Tags:        []string{"tag1"},
					PassingOnly: true,
				},
				Watch{
					Service:     "foobar",
					Tags:        []string{"tag1"},
					PassingOnly: true,
				}, // duplicates should be ignored
			},
		}
	)

	clientFactory.On("NewClient", mock.MatchedBy(func(*api.Client) bool { return true })).Return(client, ttlUpdater).Once()

	client.On("Service",
		"foobar",
		"tag1",
		true,
		mock.MatchedBy(func(qo *api.QueryOptions) bool { return qo != nil }),
	).Return([]*api.ServiceEntry{}, new(api.QueryMeta), error(nil))

	client.On("Register",
		mock.MatchedBy(func(r *api.AgentServiceRegistration) bool {
			return r.Address == "grubly.com" && r.Port == 1111
		}),
	).Return(error(nil)).Once()

	client.On("Deregister",
		mock.MatchedBy(func(r *api.AgentServiceRegistration) bool {
			return r.Address == "grubly.com" && r.Port == 1111
		}),
	).Return(error(nil)).Twice()

	e, err := NewEnvironment(logger, "", co)
	require.NoError(err)
	require.NotNil(e)

	_, ok := e.(Environment)
	assert.True(ok)

	e.Register()
	e.Deregister()

	assert.NoError(e.Close())

	clientFactory.AssertExpectations(t)
	client.AssertExpectations(t)
	ttlUpdater.AssertExpectations(t)
}

func TestNewEnvironment(t *testing.T) {
	t.Run("Empty", testNewEnvironmentEmpty)
	t.Run("ClientError", testNewEnvironmentClientError)
	t.Run("Full", testNewEnvironmentFull)
}

package service

import (
	"errors"
	"testing"

	zkclient "github.com/samuel/go-zookeeper/zk"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd"
	"github.com/go-kit/kit/sd/zk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func testZkFacade(t *testing.T, o *Options) {
	defer resetZkClientFactory()

	var (
		assert  = assert.New(t)
		require = require.New(t)
		client  = new(mockClient)

		clientEvents     = make(chan zkclient.Event, 1)
		initialInstances = []string{"instance1", "instance2"}

		instanceEvents = make(chan sd.Event, 1)
	)

	zkClientFactory = func(servers []string, logger log.Logger, options ...zk.Option) (zk.Client, error) {
		assert.Equal(o.servers(), servers)
		assert.NotNil(logger)
		assert.NotEmpty(options)
		return client, nil
	}

	if len(o.registration()) > 0 {
		client.On("Register", mock.MatchedBy(func(s *zk.Service) bool {
			assert.Equal(o.path(), s.Path)
			assert.Equal(o.serviceName(), s.Name)
			assert.Equal(o.registration(), string(s.Data))
			return true
		})).Return(error(nil)).Once()

		client.On("Deregister", mock.MatchedBy(func(s *zk.Service) bool {
			assert.Equal(o.path(), s.Path)
			assert.Equal(o.serviceName(), s.Name)
			assert.Equal(o.registration(), string(s.Data))
			return true
		})).Return(error(nil)).Twice() // once during Register/Degister, and once during Stop
	}

	client.On("CreateParentNodes", o.path()).Return(error(nil)).Once()
	client.On("GetEntries", o.path()).Return(initialInstances, (<-chan zkclient.Event)(clientEvents), error(nil)).Once()
	client.On("Stop").Once()

	service, err := New(o)
	require.NotNil(service)
	require.NoError(err)

	service.Register()
	service.Deregister()

	i, err := service.NewInstancer()
	require.NotNil(i)
	assert.NoError(err)

	i.Register(instanceEvents)
	assert.Equal(sd.Event{Instances: initialInstances}, <-instanceEvents)
	i.Deregister(instanceEvents)

	// need to do this to terminate the goroutine
	i.(*zk.Instancer).Stop()

	assert.NoError(service.Close())
	assert.NoError(service.Close()) // idempotency

	client.AssertExpectations(t)
}

func testZkFacadeClientFactoryError(t *testing.T) {
	defer resetZkClientFactory()

	var (
		assert        = assert.New(t)
		expectedError = errors.New("expected")
	)

	zkClientFactory = func([]string, log.Logger, ...zk.Option) (zk.Client, error) {
		return nil, expectedError
	}

	service, err := New(nil)
	assert.Nil(service)
	assert.Equal(expectedError, err)
}

func TestZkFacade(t *testing.T) {
	t.Run("Nil", func(t *testing.T) { testZkFacade(t, nil) })
	t.Run("Default", func(t *testing.T) { testZkFacade(t, new(Options)) })
	t.Run("Nontrivial", func(t *testing.T) {
		testZkFacade(t, &Options{
			Connection:   "host1:2181,host2:2181",
			Path:         "/foo/bar",
			ServiceName:  "testing",
			Registration: "localhost:1400",
		})
	})

	t.Run("ClientFactoryError", testZkFacadeClientFactoryError)
}

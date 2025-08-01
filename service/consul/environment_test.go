// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package consul

import (
	"reflect"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/webpa-common/v2/adapter"

	"github.com/xmidt-org/webpa-common/v2/service"
)

func testNewEnvironmentEmpty(t *testing.T) {
	defer resetClientFactory()

	var (
		assert        = assert.New(t)
		clientFactory = prepareMockClientFactory()
	)

	e, err := NewEnvironment(nil, "http", Options{})
	assert.Nil(e)
	assert.Equal(service.ErrIncomplete, err)

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

		logger        = adapter.DefaultLogger()
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

func testVerifyEnviromentRegistrars(t *testing.T) {
	var (
		require       = require.New(t)
		logger        = adapter.DefaultLogger()
		clientFactory = prepareMockClientFactory()
		client        = new(mockClient)
		ttlUpdater    = new(mockTTLUpdater)
		registrations = []api.AgentServiceRegistration{
			{
				ID:      "deadbeef",
				Name:    "deadbeef",
				Tags:    []string{"role=deadbeef, region=1"},
				Address: "deadbeef.com",
				Port:    8080,
			},
			{
				ID:      "api:deadbeef",
				Name:    "deadbeef-api",
				Tags:    []string{"role=deadbeef, region=1"},
				Address: "deadbeef.com",
				Port:    443,
			},
		}
		co = Options{
			Client: &api.Config{
				Address: "localhost:8500",
				Scheme:  "https",
			},
			Registrations: registrations,
		}
	)

	clientFactory.On("NewClient", mock.MatchedBy(func(*api.Client) bool { return true })).Return(client, ttlUpdater).Once()

	client.On("Register", mock.MatchedBy(func(r *api.AgentServiceRegistration) bool {
		return reflect.DeepEqual(registrations[0], *r)
	})).Return(error(nil)).Once()
	client.On("Register", mock.MatchedBy(func(r *api.AgentServiceRegistration) bool {
		return reflect.DeepEqual(registrations[1], *r)
	})).Return(error(nil)).Once()

	client.On("Deregister", mock.MatchedBy(func(r *api.AgentServiceRegistration) bool {
		return reflect.DeepEqual(registrations[0], *r)
	})).Return(error(nil)).Once()
	client.On("Deregister", mock.MatchedBy(func(r *api.AgentServiceRegistration) bool {
		return reflect.DeepEqual(registrations[1], *r)
	})).Return(error(nil)).Once()
	consulEnv, err := NewEnvironment(logger, service.DefaultScheme, co)
	require.NoError(err)

	consulEnv.Register()
	consulEnv.Deregister()
	clientFactory.AssertExpectations(t)
	ttlUpdater.AssertExpectations(t)
	client.AssertExpectations(t)
}

func TestNewEnvironment(t *testing.T) {
	t.Run("Empty", testNewEnvironmentEmpty)
	t.Run("ClientError", testNewEnvironmentClientError)
	t.Run("Full", testNewEnvironmentFull)
	t.Run("Verify Multi Registrar Environment", testVerifyEnviromentRegistrars)
}

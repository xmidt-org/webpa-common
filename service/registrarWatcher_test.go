package service

import (
	"github.com/strava/go.serversets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewRegistrarWatcher(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	oldBaseDirectory := serversets.BaseDirectory
	oldMemberPrefix := serversets.MemberPrefix
	defer func() {
		// cleanup
		serversets.BaseDirectory = oldBaseDirectory
		serversets.MemberPrefix = oldMemberPrefix
	}()

	testData := []*Options{
		nil,
		&Options{},
		&Options{
			BaseDirectory: "/testNewRegistrarWatcher",
			MemberPrefix:  "test_",
		},
	}

	for _, o := range testData {
		t.Logf("%v", o)
		serversets.BaseDirectory = oldBaseDirectory
		serversets.MemberPrefix = oldMemberPrefix

		registrar := NewRegistrarWatcher(o)
		require.NotNil(registrar)
		serverSet, ok := registrar.(*serversets.ServerSet)
		require.NotNil(serverSet)
		require.True(ok)

		assert.Equal(o.baseDirectory(), serversets.BaseDirectory)
		assert.Equal(o.memberPrefix(), serversets.MemberPrefix)
		assert.Equal(o.servers(), serverSet.ZookeeperServers())
		assert.Equal(o.timeout(), serverSet.ZKTimeout)
	}
}

func TestRegisterAllNoRegistrations(t *testing.T) {
	assert := assert.New(t)
	for _, o := range []*Options{nil, new(Options)} {
		t.Log(o)

		mockRegistrar := new(mockRegistrar)

		actualEndpoints, err := RegisterAll(mockRegistrar, o)
		assert.Empty(actualEndpoints)
		assert.NoError(err)

		mockRegistrar.AssertExpectations(t)
	}
}

func TestRegisterAll(t *testing.T) {
	assert := assert.New(t)
	testData := []struct {
		options           *Options
		expectedHosts     []string
		expectedPorts     []int
		expectedEndpoints []*serversets.Endpoint
		expectsError      bool
	}{
		{
			options: &Options{
				Registrations: []string{"https://node1.comcast.net:1467"},
			},
			expectedHosts:     []string{"https://node1.comcast.net"},
			expectedPorts:     []int{1467},
			expectedEndpoints: []*serversets.Endpoint{new(serversets.Endpoint)},
			expectsError:      false,
		},
		{
			options: &Options{
				Registrations: []string{"https://port.is.too.large:23987928374312"},
			},
			expectedHosts:     []string{},
			expectedPorts:     []int{},
			expectedEndpoints: nil,
			expectsError:      true,
		},
		{
			options: &Options{
				Registrations: []string{"node17.foobar.com", "https://node1.comcast.net:1467"},
			},
			expectedHosts:     []string{"http://node17.foobar.com", "https://node1.comcast.net"},
			expectedPorts:     []int{80, 1467},
			expectedEndpoints: []*serversets.Endpoint{new(serversets.Endpoint), new(serversets.Endpoint)},
			expectsError:      false,
		},
		{
			options: &Options{
				Registrations: []string{"node17.foobar.com", "https://port.is.too.large:23987928374312"},
			},
			expectedHosts:     []string{},
			expectedPorts:     []int{},
			expectedEndpoints: nil,
			expectsError:      true,
		},
		{
			options: &Options{
				Registrations: []string{"https://port.is.too.large:23987928374312", "http://valid.com:1111"},
			},
			expectedHosts:     []string{},
			expectedPorts:     []int{},
			expectedEndpoints: nil,
			expectsError:      true,
		},
	}

	for _, record := range testData {
		t.Logf("%v", record)

		mockRegistrar := new(mockRegistrar)
		for index, expectedHost := range record.expectedHosts {
			mockRegistrar.On(
				"RegisterEndpoint", expectedHost, record.expectedPorts[index], mock.AnythingOfType("func() error"),
			).Return(record.expectedEndpoints[index], nil).Once()
		}

		actualEndpoints, err := RegisterAll(mockRegistrar, record.options)
		assert.Equal(record.expectedEndpoints, actualEndpoints)
		assert.Equal(record.expectsError, err != nil)

		mockRegistrar.AssertExpectations(t)
	}
}

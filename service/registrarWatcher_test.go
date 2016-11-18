package service

import (
	"errors"
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

func TestParseRegistration(t *testing.T) {
	assert := assert.New(t)
	testData := []struct {
		registration string
		expectedHost string
		expectedPort uint16
		expectsError bool
	}{
		{
			registration: "localhost",
			expectedHost: "http://localhost",
			expectedPort: 80,
			expectsError: false,
		},
		{
			registration: "foobar.com",
			expectedHost: "http://foobar.com",
			expectedPort: 80,
			expectsError: false,
		},
		{
			registration: "http://foobar.com",
			expectedHost: "http://foobar.com",
			expectedPort: 80,
			expectsError: false,
		},
		{
			registration: "https://foobar.com",
			expectedHost: "https://foobar.com",
			expectedPort: 443,
			expectsError: false,
		},
		{
			registration: "https://node1.webpa.comcast.net:1847",
			expectedHost: "https://node1.webpa.comcast.net",
			expectedPort: 1847,
			expectsError: false,
		},
		{
			registration: "node1.webpa.comcast.net:8080",
			expectedHost: "http://node1.webpa.comcast.net",
			expectedPort: 8080,
			expectsError: false,
		},
		{
			registration: "something.webpa.comcast.net:0",
			expectedHost: "http://something.webpa.comcast.net",
			expectedPort: 80,
			expectsError: false,
		},
		{
			registration: "http://something.webpa.comcast.net:0",
			expectedHost: "http://something.webpa.comcast.net",
			expectedPort: 80,
			expectsError: false,
		},
		{
			registration: "https://something.webpa.comcast.net:0",
			expectedHost: "https://something.webpa.comcast.net",
			expectedPort: 443,
			expectsError: false,
		},
		{
			registration: "unrecognized://something.webpa.comcast.net",
			expectedHost: "unrecognized://something.webpa.comcast.net",
			expectedPort: 0,
			expectsError: false,
		},
		{
			registration: "unrecognized://something.webpa.comcast.net:0",
			expectedHost: "unrecognized://something.webpa.comcast.net",
			expectedPort: 0,
			expectsError: false,
		},
		{
			registration: "port.is.too.large.net:35982739476",
			expectedHost: "http://port.is.too.large.net",
			expectedPort: 0,
			expectsError: true,
		},
	}

	for _, record := range testData {
		t.Logf("%v", record)

		actualHost, actualPort, err := ParseRegistration(record.registration)
		assert.Equal(record.expectedHost, actualHost)
		assert.Equal(record.expectedPort, actualPort)
		assert.Equal(record.expectsError, err != nil)
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

func TestRegisterAllEndpointFailure(t *testing.T) {
	assert := assert.New(t)
	expectedError := errors.New("expected endpoint error")
	options := &Options{
		Registrations: []string{"node1.comcast.net:8080"},
	}

	mockRegistrar := new(mockRegistrar)
	mockRegistrar.On("RegisterEndpoint", "http://node1.comcast.net", 8080, mock.AnythingOfType("func() error")).
		Return(nil, expectedError).
		Once()

	actualEndpoints, err := RegisterAll(mockRegistrar, options)
	assert.Empty(actualEndpoints)
	assert.Equal(expectedError, err)

	mockRegistrar.AssertExpectations(t)
}

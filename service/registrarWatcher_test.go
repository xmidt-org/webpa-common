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

var (
	singleRegistration = Registration{
		Scheme: "https",
		Host:   "foobar123.webpa.comcast.net",
		Port:   8080,
	}
)

const (
	expectedRegistrationHost = "https://foobar123.webpa.comcast.net"
	expectedRegistrationPort = 8080
)

func TestRegisterOneSuccessWithoutPingFunction(t *testing.T) {
	assert := assert.New(t)
	expectedEndpoint := new(serversets.Endpoint)
	mockRegistrar := new(mockRegistrar)
	mockRegistrar.
		On("RegisterEndpoint", expectedRegistrationHost, expectedRegistrationPort, mock.AnythingOfType("func() error")).
		Return(expectedEndpoint, nil).
		Once()

	actualEndpoint, err := RegisterOne(mockRegistrar, singleRegistration, nil)
	assert.Equal(expectedEndpoint, actualEndpoint)
	assert.NoError(err)

	mockRegistrar.AssertExpectations(t)
}

func TestRegisterOneFailureWithoutPingFunction(t *testing.T) {
	assert := assert.New(t)
	expectedRegisterError := errors.New("expected register error")
	expectedEndpoint := new(serversets.Endpoint)
	mockRegistrar := new(mockRegistrar)
	mockRegistrar.
		On("RegisterEndpoint", expectedRegistrationHost, expectedRegistrationPort, mock.AnythingOfType("func() error")).
		Return(expectedEndpoint, expectedRegisterError).
		Once()

	actualEndpoint, err := RegisterOne(mockRegistrar, singleRegistration, nil)
	assert.Equal(expectedEndpoint, actualEndpoint)
	assert.Equal(expectedRegisterError, err)

	mockRegistrar.AssertExpectations(t)
}

func TestRegisterOneSuccessWithPingFunction(t *testing.T) {
	assert := assert.New(t)
	expectedPingError := errors.New("expected ping error")
	expectedEndpoint := new(serversets.Endpoint)
	mockRegistrar := new(mockRegistrar)
	mockRegistrar.
		On("RegisterEndpoint", expectedRegistrationHost, expectedRegistrationPort, mock.AnythingOfType("func() error")).
		Run(func(arguments mock.Arguments) {
			assert.Equal(expectedPingError, arguments.Get(2).(func() error)())
		}).
		Return(expectedEndpoint, nil).
		Once()

	actualEndpoint, err := RegisterOne(mockRegistrar, singleRegistration, func() error { return expectedPingError })
	assert.Equal(expectedEndpoint, actualEndpoint)
	assert.NoError(err)

	mockRegistrar.AssertExpectations(t)
}

func TestRegisterOneFailureWithPingFunction(t *testing.T) {
	assert := assert.New(t)
	expectedRegisterError := errors.New("expected register error")
	expectedPingError := errors.New("expected ping error")
	expectedEndpoint := new(serversets.Endpoint)
	mockRegistrar := new(mockRegistrar)
	mockRegistrar.
		On("RegisterEndpoint", expectedRegistrationHost, expectedRegistrationPort, mock.AnythingOfType("func() error")).
		Run(func(arguments mock.Arguments) {
			assert.Equal(expectedPingError, arguments.Get(2).(func() error)())
		}).
		Return(expectedEndpoint, expectedRegisterError).
		Once()

	actualEndpoint, err := RegisterOne(mockRegistrar, singleRegistration, func() error { return expectedPingError })
	assert.Equal(expectedEndpoint, actualEndpoint)
	assert.Equal(err, expectedRegisterError)

	mockRegistrar.AssertExpectations(t)
}

func TestRegisterOneBadPort(t *testing.T) {
	assert := assert.New(t)
	badPortRegistration := Registration{
		Scheme: "unrecognized",
		Host:   "foobar.com",
	}

	mockRegistrar := new(mockRegistrar)

	actualEndpoint, err := RegisterOne(mockRegistrar, badPortRegistration, nil)
	assert.Nil(actualEndpoint)
	assert.Error(err)

	mockRegistrar.AssertExpectations(t)
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
				Registrations: []Registration{
					Registration{Scheme: "https", Host: "node1.comcast.net", Port: 1467},
				},
			},
			expectedHosts:     []string{"https://node1.comcast.net"},
			expectedPorts:     []int{1467},
			expectedEndpoints: []*serversets.Endpoint{new(serversets.Endpoint)},
			expectsError:      false,
		},
		{
			options: &Options{
				Registrations: []Registration{
					Registration{Scheme: "unregonized", Host: "this.should.not.be.registered.com"},
				},
			},
			expectedHosts:     []string{},
			expectedPorts:     []int{},
			expectedEndpoints: []*serversets.Endpoint{},
			expectsError:      true,
		},
		{
			options: &Options{
				Registrations: []Registration{
					Registration{Host: "node17.foobar.com"},
					Registration{Scheme: "https", Host: "node1.comcast.net", Port: 1467},
				},
			},
			expectedHosts:     []string{"http://node17.foobar.com", "https://node1.comcast.net"},
			expectedPorts:     []int{80, 1467},
			expectedEndpoints: []*serversets.Endpoint{new(serversets.Endpoint), new(serversets.Endpoint)},
			expectsError:      false,
		},
		{
			options: &Options{
				Registrations: []Registration{
					Registration{Scheme: "unregonized", Host: "this.should.not.be.registered.com"},
					Registration{Host: "node17.foobar.com"},
					Registration{Scheme: "https", Host: "node1.comcast.net", Port: 1467},
				},
			},
			expectedHosts:     []string{},
			expectedPorts:     []int{},
			expectedEndpoints: []*serversets.Endpoint{},
			expectsError:      true,
		},
		{
			options: &Options{
				Registrations: []Registration{
					Registration{Host: "node17.foobar.com"},
					Registration{Scheme: "unregonized", Host: "this.should.not.be.registered.com"},
					Registration{Scheme: "https", Host: "node1.comcast.net", Port: 1467},
				},
			},
			expectedHosts:     []string{"http://node17.foobar.com"},
			expectedPorts:     []int{80},
			expectedEndpoints: []*serversets.Endpoint{new(serversets.Endpoint)},
			expectsError:      true,
		},
		{
			options: &Options{
				Registrations: []Registration{
					Registration{Host: "node17.foobar.com"},
					Registration{Scheme: "https", Host: "node1.comcast.net", Port: 1467},
					Registration{Scheme: "unregonized", Host: "this.should.not.be.registered.com"},
				},
			},
			expectedHosts:     []string{"http://node17.foobar.com", "https://node1.comcast.net"},
			expectedPorts:     []int{80, 1467},
			expectedEndpoints: []*serversets.Endpoint{new(serversets.Endpoint), new(serversets.Endpoint)},
			expectsError:      true,
		},
		{
			options: &Options{
				Registrations: []Registration{
					Registration{Host: "node17.foobar.com"},
					Registration{Scheme: "https", Host: "node1.comcast.net", Port: 1467},
					Registration{Scheme: "http", Host: "gronk.something.tv", Port: 610},
				},
			},
			expectedHosts:     []string{"http://node17.foobar.com", "https://node1.comcast.net", "http://gronk.something.tv"},
			expectedPorts:     []int{80, 1467, 610},
			expectedEndpoints: []*serversets.Endpoint{new(serversets.Endpoint), new(serversets.Endpoint), new(serversets.Endpoint)},
			expectsError:      false,
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

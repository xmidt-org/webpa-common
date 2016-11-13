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
		assert.Equal(o.zookeeperServers(), serverSet.ZookeeperServers())
		assert.Equal(o.zookeeperTimeout(), serverSet.ZKTimeout)
	}
}

var registrationTestData = []struct {
	registration Registration
	expectedHost string
	expectedPort int
}{
	{
		Registration{},
		"http://localhost",
		80,
	},
}

func TestRegisterOneSuccessAndNilPingFunc(t *testing.T) {
	assert := assert.New(t)

	for _, record := range registrationTestData {
		t.Logf("%v", record)
		assert.True(true)

		expectedEndpoint := new(serversets.Endpoint)
		mockRegistrar := new(mockRegistrar)
		mockRegistrar.
			On("RegisterEndpoint", record.expectedHost, record.expectedPort, mock.AnythingOfType("func() error")).
			Return(expectedEndpoint, nil).
			Once()

		actualEndpoint, err := RegisterOne(record.registration, nil, mockRegistrar)
		assert.Equal(expectedEndpoint, actualEndpoint)
		assert.NoError(err)

		mockRegistrar.AssertExpectations(t)
	}
}

package service

import (
	"github.com/strava/go.serversets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

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

func TestRegisterWithSuccessAndNilPingFunc(t *testing.T) {
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

		actualEndpoint, err := RegisterWith(record.registration, nil, mockRegistrar)
		assert.Equal(expectedEndpoint, actualEndpoint)
		assert.NoError(err)

		mockRegistrar.AssertExpectations(t)
	}
}

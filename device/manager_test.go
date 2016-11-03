package device

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestManagerConnectMissingDeviceNameHeader(t *testing.T) {
	assert := assert.New(t)

	manager := NewManager(nil, nil)
	response := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "http://localhost.com", nil)

	device, err := manager.Connect(response, request, nil)
	assert.Nil(device)
	assert.NotNil(err)
	assert.Equal(response.Code, http.StatusBadRequest)
}

func TestManagerBadDeviceNameHeader(t *testing.T) {
	assert := assert.New(t)

	manager := NewManager(nil, nil)
	response := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "http://localhost.com", nil)
	request.Header.Set(DefaultDeviceNameHeader, "this is not valid")

	device, err := manager.Connect(response, request, nil)
	assert.Nil(device)
	assert.NotNil(err)
	assert.Equal(response.Code, http.StatusBadRequest)
}

func TestManagerBadConveyHeader(t *testing.T) {
	assert := assert.New(t)

	manager := NewManager(nil, nil)
	response := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "http://localhost.com", nil)
	request.Header.Set(DefaultDeviceNameHeader, "mac:112233445566")
	request.Header.Set(DefaultConveyHeader, "this is not valid")

	device, err := manager.Connect(response, request, nil)
	assert.Nil(device)
	assert.NotNil(err)
	assert.Equal(response.Code, http.StatusBadRequest)
}

func TestManagerKeyError(t *testing.T) {
	assert := assert.New(t)

	badKeyFunc := func(ID, *Convey, *http.Request) (Key, error) {
		return invalidKey, errors.New("expected")
	}

	manager := NewManager(&Options{KeyFunc: badKeyFunc}, nil)
	response := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "http://localhost.com", nil)
	request.Header.Set(DefaultDeviceNameHeader, "mac:112233445566")

	device, err := manager.Connect(response, request, nil)
	assert.Nil(device)
	assert.NotNil(err)
	assert.Equal(response.Code, http.StatusBadRequest)
}

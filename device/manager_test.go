package device

import (
	"errors"
	"fmt"
	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// startWebsocketServer sets up a server-side environment for testing device-related websocket code
func startWebsocketServer(o *Options) (Manager, *httptest.Server, string) {
	manager := NewManager(o, nil)
	server := httptest.NewServer(
		NewConnectHandler(
			manager,
			nil,
			o.logger(),
		),
	)

	websocketURL, err := url.Parse(server.URL)
	if err != nil {
		server.Close()
		panic(fmt.Errorf("Unable to parse test server URL: %s", err))
	}

	websocketURL.Scheme = "ws"
	return manager, server, websocketURL.String()
}

func TestManagerConnectMissingDeviceNameHeader(t *testing.T) {
	assert := assert.New(t)
	options := &Options{
		Logger: logging.TestLogger(t),
	}

	manager := NewManager(options, nil)
	response := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "http://localhost.com", nil)

	device, err := manager.Connect(response, request, nil)
	assert.Nil(device)
	assert.NotNil(err)
	assert.Equal(response.Code, http.StatusBadRequest)
}

func TestManagerBadDeviceNameHeader(t *testing.T) {
	assert := assert.New(t)
	options := &Options{
		Logger: logging.TestLogger(t),
	}

	manager := NewManager(options, nil)
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
	options := &Options{
		Logger: logging.TestLogger(t),
	}

	manager := NewManager(options, nil)
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
	badKeyFunc := func(ID, Convey, *http.Request) (Key, error) {
		return invalidKey, errors.New("expected")
	}

	options := &Options{
		Logger:  logging.TestLogger(t),
		KeyFunc: badKeyFunc,
	}

	manager := NewManager(options, nil)
	response := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "http://localhost.com", nil)
	request.Header.Set(DefaultDeviceNameHeader, "mac:112233445566")

	device, err := manager.Connect(response, request, nil)
	assert.Nil(device)
	assert.NotNil(err)
	assert.Equal(response.Code, http.StatusBadRequest)
}

func TestManager(t *testing.T) {
	assert := assert.New(t)
	options := &Options{
		Logger: logging.TestLogger(t),
	}

	_, server, connectURL := startWebsocketServer(options)
	defer server.Close()

	dialer := NewDialer(options, nil)

	singleID1 := IntToMAC(0xFEA78CBA190B)
	singleConnection1, _, err := dialer.Dial(connectURL, singleID1, nil, nil)
	if !assert.Nil(err) {
		t.FailNow()
	}

	defer singleConnection1.Close()
}

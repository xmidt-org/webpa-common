package device

import (
	"errors"
	"fmt"
	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
)

var (
	testDeviceIDs = map[ID]int{
		IntToMAC(0xDEADBEEF):     1,
		IntToMAC(0x112233445566): 1,
		IntToMAC(0xFE881212CDCD): 2,
		IntToMAC(0x7F551928ABCD): 3,
	}

	testConnectionCount = func() (total int) {
		for _, connectionCount := range testDeviceIDs {
			total += connectionCount
		}

		return
	}()
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

func connectTestDevices(t *testing.T, assert *assert.Assertions, dialer Dialer, connectURL string) map[ID][]Connection {
	devices := make(map[ID][]Connection, len(testDeviceIDs))

	for id, connectionCount := range testDeviceIDs {
		connections := make([]Connection, 0, connectionCount)
		for repeat := 0; repeat < connectionCount; repeat++ {
			deviceConnection, response, err := dialer.Dial(connectURL, id, nil, nil)
			if assert.NotNil(deviceConnection) && assert.NotNil(response) && assert.Nil(err) {
				connections = append(connections, deviceConnection)
			} else {
				t.FailNow()
			}
		}

		devices[id] = connections
	}

	return devices
}

func closeTestDevices(assert *assert.Assertions, devices map[ID][]Connection) {
	for _, connections := range devices {
		for _, connection := range connections {
			assert.Nil(connection.Close())
		}
	}
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

func TestManagerPongCallbackFor(t *testing.T) {
	assert := assert.New(t)
	expectedDevice := newDevice(ID("ponged device"), Key("expected"), nil, 1)
	expectedData := "expected pong data"
	listenerCalled := false

	manager := &manager{
		logger: logging.TestLogger(t),
		pongListener: func(actualDevice Interface, actualData string) {
			listenerCalled = true
			assert.True(expectedDevice == actualDevice)
			assert.Equal(expectedData, actualData)
		},
	}

	pongCallback := manager.pongCallbackFor(expectedDevice)
	pongCallback(expectedData)
	assert.True(listenerCalled)
}

func TestManagerConnectAndVisit(t *testing.T) {
	assert := assert.New(t)
	connectWait := new(sync.WaitGroup)
	connectWait.Add(testConnectionCount)
	connections := make(chan Interface, testConnectionCount)

	options := &Options{
		Logger: logging.TestLogger(t),
		ConnectListener: func(candidate Interface) {
			defer connectWait.Done()
			select {
			case connections <- candidate:
			default:
				assert.Fail("The connect listener should not block")
			}
		},
	}

	manager, server, connectURL := startWebsocketServer(options)
	defer server.Close()

	dialer := NewDialer(options, nil)
	testDevices := connectTestDevices(t, assert, dialer, connectURL)
	defer closeTestDevices(assert, testDevices)

	connectWait.Wait()
	close(connections)
	assert.Equal(testConnectionCount, len(connections))

	deviceSet := make(deviceSet)
	for candidate := range connections {
		deviceSet.add(candidate)
	}

	assert.Equal(testConnectionCount, deviceSet.len())

	deviceSet.reset()
	assert.Zero(manager.VisitIf(
		func(ID) bool { return false },
		deviceSet.managerCapture(),
	))

	assert.Empty(deviceSet)

	for id, connectionCount := range testDeviceIDs {
		deviceSet.reset()
		assert.Equal(
			connectionCount,
			manager.VisitIf(
				func(candidate ID) bool { return candidate == id },
				deviceSet.managerCapture(),
			),
		)

		assert.Equal(connectionCount, deviceSet.len())
		deviceSet.assertSameID(assert, id)
	}

	deviceSet.reset()
	manager.VisitAll(deviceSet.managerCapture())
	deviceSet.assertDistributionOfIDs(assert, testDeviceIDs)
}

func TestManagerDisconnect(t *testing.T) {
	assert := assert.New(t)
	disconnectWait := new(sync.WaitGroup)
	disconnectWait.Add(testConnectionCount)
	disconnections := make(chan Interface, testConnectionCount)

	options := &Options{
		Logger: logging.TestLogger(t),
		DisconnectListener: func(candidate Interface) {
			defer disconnectWait.Done()
			assert.True(candidate.Closed())
			disconnections <- candidate
		},
	}

	manager, server, connectURL := startWebsocketServer(options)
	defer server.Close()

	dialer := NewDialer(options, nil)
	testDevices := connectTestDevices(t, assert, dialer, connectURL)
	defer closeTestDevices(assert, testDevices)

	assert.Zero(manager.Disconnect(ID("nosuch")))
	for id, connectionCount := range testDeviceIDs {
		assert.Equal(connectionCount, manager.Disconnect(id))
	}

	disconnectWait.Wait()
	close(disconnections)
	assert.Equal(testConnectionCount, len(disconnections))

	visited := make(map[Interface]bool)
	for candidate := range disconnections {
		visited[candidate] = true
	}

	assert.Equal(testConnectionCount, len(visited))
}

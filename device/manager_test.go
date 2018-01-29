package device

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/wrp"
	"github.com/justinas/alice"
	"github.com/stretchr/testify/assert"
)

var (
	testDeviceIDs = []ID{
		IntToMAC(0xDEADBEEF),
		IntToMAC(0x112233445566),
		IntToMAC(0xFE881212CDCD),
		IntToMAC(0x7F551928ABCD),
	}
)

// startWebsocketServer sets up a server-side environment for testing device-related websocket code
func startWebsocketServer(o *Options) (Manager, *httptest.Server, string) {
	var (
		manager = NewManager(o)
		server  = httptest.NewServer(
			alice.New(Timeout(o), UseID.FromHeader).Then(
				&ConnectHandler{
					Logger:    o.logger(),
					Connector: manager,
				},
			),
		)

		websocketURL, err = url.Parse(server.URL)
	)

	if err != nil {
		server.Close()
		panic(fmt.Errorf("Unable to parse test server URL: %s", err))
	}

	websocketURL.Scheme = "ws"
	return manager, server, websocketURL.String()
}

func connectTestDevices(t *testing.T, dialer Dialer, connectURL string) map[ID]Connection {
	devices := make(map[ID]Connection, len(testDeviceIDs))

	for _, id := range testDeviceIDs {
		deviceConnection, _, err := dialer.DialDevice(string(id), connectURL, nil)
		if err != nil {
			t.Fatalf("Unable to dial test device: %s", err)
			break
		}

		devices[id] = deviceConnection
	}

	return devices
}

func closeTestDevices(assert *assert.Assertions, devices map[ID]Connection) {
	for _, connection := range devices {
		assert.Nil(connection.Close())
	}
}

func testManagerConnectMissingDeviceContext(t *testing.T) {
	assert := assert.New(t)
	options := &Options{
		Logger: logging.NewTestLogger(nil, t),
	}

	manager := NewManager(options)
	response := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "http://localhost.com", nil)

	device, err := manager.Connect(response, request, nil)
	assert.Nil(device)
	assert.Error(err)
	assert.Equal(response.Code, http.StatusInternalServerError)
}

func testManagerConnectConnectionFactoryError(t *testing.T) {
	var (
		assert  = assert.New(t)
		options = &Options{
			Logger: logging.NewTestLogger(nil, t),
			Listeners: []Listener{
				func(e *Event) {
					assert.Fail("The listener should not have been called")
				},
			},
		}

		manager        = NewManager(options)
		response       = httptest.NewRecorder()
		request        = WithIDRequest(ID("mac:123412341234"), httptest.NewRequest("POST", "http://localhost.com", nil))
		responseHeader http.Header
		expectedError  = errors.New("expected error")
	)

	device, actualError := manager.Connect(response, request, responseHeader)
	assert.Nil(device)
	assert.Equal(expectedError, actualError)
}

func testManagerConnectVisit(t *testing.T) {
	var (
		assert      = assert.New(t)
		connectWait = new(sync.WaitGroup)
		connections = make(chan Interface, len(testDeviceIDs))

		options = &Options{
			Logger: logging.NewTestLogger(nil, t),
			Listeners: []Listener{
				func(event *Event) {
					if event.Type == Connect {
						defer connectWait.Done()
						select {
						case connections <- event.Device:
						default:
							assert.Fail("The connect listener should not block")
						}
					}
				},
			},
		}

		manager, server, connectURL = startWebsocketServer(options)
	)

	defer server.Close()
	connectWait.Add(len(testDeviceIDs))

	testDevices := connectTestDevices(t, DefaultDialer(), connectURL)
	defer closeTestDevices(assert, testDevices)

	connectWait.Wait()
	close(connections)
	assert.Equal(len(testDeviceIDs), len(connections))

	deviceSet := make(deviceSet)
	for candidate := range connections {
		deviceSet.add(candidate)
	}

	assert.Equal(len(testDeviceIDs), deviceSet.len())
	deviceSet.reset()
	manager.VisitAll(deviceSet.managerCapture())
	assert.Equal(len(testDeviceIDs), deviceSet.len())
}

func testManagerDisconnect(t *testing.T) {
	assert := assert.New(t)
	connectWait := new(sync.WaitGroup)
	connectWait.Add(len(testDeviceIDs))

	disconnectWait := new(sync.WaitGroup)
	disconnectWait.Add(len(testDeviceIDs))
	disconnections := make(chan Interface, len(testDeviceIDs))

	options := &Options{
		Logger: logging.NewTestLogger(nil, t),
		Listeners: []Listener{
			func(event *Event) {
				switch event.Type {
				case Connect:
					connectWait.Done()
				case Disconnect:
					defer disconnectWait.Done()
					assert.True(event.Device.Closed())
					disconnections <- event.Device
				}
			},
		},
	}

	manager, server, connectURL := startWebsocketServer(options)
	defer server.Close()

	testDevices := connectTestDevices(t, DefaultDialer(), connectURL)
	defer closeTestDevices(assert, testDevices)

	connectWait.Wait()
	assert.Zero(manager.Disconnect(ID("nosuch")))
	for _, id := range testDeviceIDs {
		assert.Equal(1, manager.Disconnect(id))
	}

	disconnectWait.Wait()
	close(disconnections)
	assert.Equal(len(testDeviceIDs), len(disconnections))

	deviceSet := make(deviceSet)
	deviceSet.drain(disconnections)
	assert.Equal(len(testDeviceIDs), deviceSet.len())
}

func testManagerDisconnectIf(t *testing.T) {
	assert := assert.New(t)
	connectWait := new(sync.WaitGroup)
	connectWait.Add(len(testDeviceIDs))
	disconnections := make(chan Interface, len(testDeviceIDs))

	options := &Options{
		Logger: logging.NewTestLogger(nil, t),
		Listeners: []Listener{
			func(event *Event) {
				switch event.Type {
				case Connect:
					connectWait.Done()
				case Disconnect:
					assert.True(event.Device.Closed())
					disconnections <- event.Device
				}
			},
		},
	}

	manager, server, connectURL := startWebsocketServer(options)
	defer server.Close()

	testDevices := connectTestDevices(t, DefaultDialer(), connectURL)
	defer closeTestDevices(assert, testDevices)

	connectWait.Wait()
	deviceSet := make(deviceSet)
	manager.VisitAll(deviceSet.managerCapture())
	assert.Equal(len(testDeviceIDs), deviceSet.len())

	assert.Zero(manager.DisconnectIf(func(ID) bool { return false }))
	select {
	case <-disconnections:
		assert.Fail("No disconnections should have occurred")
	default:
		// the passing case
	}

	for _, id := range testDeviceIDs {
		assert.Equal(1, manager.DisconnectIf(func(candidate ID) bool { return candidate == id }))
		select {
		case actual := <-disconnections:
			assert.Equal(id, actual.ID())
			assert.True(actual.Closed())
		case <-time.After(10 * time.Second):
			assert.Fail("No disconnection occurred within the timeout")
		}
	}
}

func testManagerRouteBadDestination(t *testing.T) {
	var (
		assert  = assert.New(t)
		request = &Request{
			Message: &wrp.Message{
				Destination: "this is a bad destination",
			},
		}

		manager = NewManager(nil)
	)

	response, err := manager.Route(request)
	assert.Nil(response)
	assert.Error(err)
}

func testManagerRouteDeviceNotFound(t *testing.T) {
	var (
		assert  = assert.New(t)
		request = &Request{
			Message: &wrp.Message{
				Destination: "mac:112233445566",
			},
		}

		manager = NewManager(nil)
	)

	response, err := manager.Route(request)
	assert.Nil(response)
	assert.Equal(ErrorDeviceNotFound, err)
}

func testManagerRouteNonUniqueID(t *testing.T) {
	var (
		assert  = assert.New(t)
		request = &Request{
			Message: &wrp.Message{
				Destination: "mac:112233445566",
			},
		}

		logger  = logging.NewTestLogger(nil, t)
		device1 = newDevice(ID("mac:112233445566"), 1, time.Now(), logger)
		device2 = newDevice(ID("mac:112233445566"), 1, time.Now(), logger)

		manager = NewManager(nil).(*manager)
	)

	manager.devices.add(device1.id, func() (*device, error) { return device1, nil })
	manager.devices.add(device2.id, func() (*device, error) { return device2, nil })

	response, err := manager.Route(request)
	assert.Nil(response)
	assert.Equal(ErrorNonUniqueID, err)
}

func TestManager(t *testing.T) {
	/*
			t.Run("Connect", func(t *testing.T) {
				t.Run("MissingDeviceContext", testManagerConnectMissingDeviceContext)
				t.Run("ConnectionFactoryError", testManagerConnectConnectionFactoryError)
				t.Run("Visit", testManagerConnectVisit)
			})

			t.Run("Route", func(t *testing.T) {
				t.Run("BadDestination", testManagerRouteBadDestination)
				t.Run("DeviceNotFound", testManagerRouteDeviceNotFound)
				t.Run("NonUniqueID", testManagerRouteNonUniqueID)
			})

		t.Run("Disconnect", testManagerDisconnect)
	*/
	t.Run("DisconnectIf", testManagerDisconnectIf)
}

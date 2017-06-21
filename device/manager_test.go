package device

import (
	"errors"
	"fmt"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/wrp"
	"github.com/justinas/alice"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"
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
	var (
		manager = NewManager(o, nil)
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

func connectTestDevices(t *testing.T, assert *assert.Assertions, dialer Dialer, connectURL string) map[ID][]Connection {
	devices := make(map[ID][]Connection, len(testDeviceIDs))

	for id, connectionCount := range testDeviceIDs {
		connections := make([]Connection, 0, connectionCount)
		for repeat := 0; repeat < connectionCount; repeat++ {
			deviceConnection, response, err := dialer.Dial(connectURL, id, nil, nil)
			if assert.NotNil(deviceConnection) && assert.NotNil(response) && assert.NoError(err) {
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

func testManagerConnectMissingDeviceContext(t *testing.T) {
	assert := assert.New(t)
	options := &Options{
		Logger: logging.TestLogger(t),
	}

	manager := NewManager(options, nil)
	response := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "http://localhost.com", nil)

	device, err := manager.Connect(response, request, nil)
	assert.Nil(device)
	assert.Error(err)
	assert.Equal(response.Code, http.StatusInternalServerError)
}

func testManagerConnectBadConveyHeader(t *testing.T) {
	assert := assert.New(t)
	options := &Options{
		Logger: logging.TestLogger(t),
	}

	manager := NewManager(options, nil)
	response := httptest.NewRecorder()
	request := WithIDRequest(ID("mac:112233445566"), httptest.NewRequest("POST", "http://localhost.com", nil))
	request.Header.Set(ConveyHeader, "this is not valid")

	device, err := manager.Connect(response, request, nil)
	assert.Nil(device)
	assert.Error(err)
	assert.Equal(response.Code, http.StatusBadRequest)
}

func testManagerConnectKeyError(t *testing.T) {
	var (
		assert     = assert.New(t)
		badKeyFunc = func(ID, Convey, *http.Request) (Key, error) {
			return invalidKey, errors.New("expected")
		}

		options = &Options{
			Logger:  logging.TestLogger(t),
			KeyFunc: badKeyFunc,
		}

		manager  = NewManager(options, nil)
		response = httptest.NewRecorder()
		request  = WithIDRequest(ID("mac:112233445566"), httptest.NewRequest("POST", "http://localhost.com", nil))
	)

	device, err := manager.Connect(response, request, nil)
	assert.Nil(device)
	assert.Error(err)
	assert.Equal(response.Code, http.StatusBadRequest)
}

func testManagerConnectConnectionFactoryError(t *testing.T) {
	var (
		assert  = assert.New(t)
		options = &Options{
			Logger: logging.TestLogger(t),
			Listeners: []Listener{
				func(e *Event) {
					assert.Fail("The listener should not have been called")
				},
			},
		}

		connectionFactory = new(mockConnectionFactory)
		manager           = NewManager(options, connectionFactory)
		response          = httptest.NewRecorder()
		request           = WithIDRequest(ID("mac:123412341234"), httptest.NewRequest("POST", "http://localhost.com", nil))
		responseHeader    http.Header
		expectedError     = errors.New("expected error")
	)

	connectionFactory.On("NewConnection", response, request, responseHeader).Once().Return(nil, expectedError)

	device, actualError := manager.Connect(response, request, responseHeader)
	assert.Nil(device)
	assert.Equal(expectedError, actualError)

	connectionFactory.AssertExpectations(t)
}

func testManagerConnectVisit(t *testing.T) {
	var (
		assert      = assert.New(t)
		connectWait = new(sync.WaitGroup)
		connections = make(chan Interface, testConnectionCount)

		options = &Options{
			Logger: logging.TestLogger(t),
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
	connectWait.Add(testConnectionCount)

	var (
		dialer      = NewDialer(options, nil)
		testDevices = connectTestDevices(t, assert, dialer, connectURL)
	)

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

func testManagerPongCallbackFor(t *testing.T) {
	assert := assert.New(t)
	expectedDevice := newDevice(ID("ponged device"), Key("expected"), nil, "", 1)
	expectedData := "expected pong data"
	listenerCalled := false

	manager := &manager{
		logger: logging.TestLogger(t),
		listeners: []Listener{
			func(event *Event) {
				listenerCalled = true
				assert.True(expectedDevice == event.Device)
				assert.Equal(expectedData, event.Data)
			},
		},
	}

	pongCallback := manager.pongCallbackFor(expectedDevice)
	pongCallback(expectedData)
	assert.True(listenerCalled)
}

func testManagerDisconnect(t *testing.T) {
	assert := assert.New(t)
	connectWait := new(sync.WaitGroup)
	connectWait.Add(testConnectionCount)

	disconnectWait := new(sync.WaitGroup)
	disconnectWait.Add(testConnectionCount)
	disconnections := make(chan Interface, testConnectionCount)

	options := &Options{
		Logger: logging.TestLogger(t),
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

	dialer := NewDialer(options, nil)
	testDevices := connectTestDevices(t, assert, dialer, connectURL)
	defer closeTestDevices(assert, testDevices)

	connectWait.Wait()
	assert.Zero(manager.Disconnect(ID("nosuch")))
	for id, connectionCount := range testDeviceIDs {
		assert.Equal(connectionCount, manager.Disconnect(id))
	}

	disconnectWait.Wait()
	close(disconnections)
	assert.Equal(testConnectionCount, len(disconnections))

	deviceSet := make(deviceSet)
	deviceSet.drain(disconnections)
	assert.Equal(testConnectionCount, deviceSet.len())
}

func testManagerDisconnectOne(t *testing.T) {
	assert := assert.New(t)
	connectWait := new(sync.WaitGroup)
	connectWait.Add(testConnectionCount)
	disconnections := make(chan Interface, testConnectionCount)

	options := &Options{
		Logger: logging.TestLogger(t),
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

	dialer := NewDialer(options, nil)
	testDevices := connectTestDevices(t, assert, dialer, connectURL)
	defer closeTestDevices(assert, testDevices)

	connectWait.Wait()
	deviceSet := make(deviceSet)
	manager.VisitAll(deviceSet.managerCapture())
	assert.Equal(testConnectionCount, deviceSet.len())

	assert.Zero(manager.DisconnectOne(Key("nosuch")))
	select {
	case <-disconnections:
		assert.Fail("No disconnections should have occurred")
	default:
		// the passing case
	}

	for expected, _ := range deviceSet {
		assert.Equal(1, manager.DisconnectOne(expected.Key()))

		select {
		case actual := <-disconnections:
			assert.Equal(expected.Key(), actual.Key())
			assert.True(actual.Closed())
		case <-time.After(10 * time.Second):
			assert.Fail("No disconnection occurred within the timeout")
		}
	}
}

func testManagerDisconnectIf(t *testing.T) {
	assert := assert.New(t)
	connectWait := new(sync.WaitGroup)
	connectWait.Add(testConnectionCount)
	disconnections := make(chan Interface, testConnectionCount)

	options := &Options{
		Logger: logging.TestLogger(t),
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

	dialer := NewDialer(options, nil)
	testDevices := connectTestDevices(t, assert, dialer, connectURL)
	defer closeTestDevices(assert, testDevices)

	connectWait.Wait()
	deviceSet := make(deviceSet)
	manager.VisitAll(deviceSet.managerCapture())
	assert.Equal(testConnectionCount, deviceSet.len())

	assert.Zero(manager.DisconnectIf(func(ID) bool { return false }))
	select {
	case <-disconnections:
		assert.Fail("No disconnections should have occurred")
	default:
		// the passing case
	}

	for id, connectionCount := range testDeviceIDs {
		assert.Equal(connectionCount, manager.DisconnectIf(func(candidate ID) bool { return candidate == id }))
		for repeat := 0; repeat < connectionCount; repeat++ {
			select {
			case actual := <-disconnections:
				assert.Equal(id, actual.ID())
				assert.True(actual.Closed())
			case <-time.After(10 * time.Second):
				assert.Fail("No disconnection occurred within the timeout")
			}
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

		connectionFactory = new(mockConnectionFactory)
		manager           = NewManager(nil, connectionFactory)
	)

	response, err := manager.Route(request)
	assert.Nil(response)
	assert.Error(err)

	connectionFactory.AssertExpectations(t)
}

func testManagerRouteDeviceNotFound(t *testing.T) {
	var (
		assert  = assert.New(t)
		request = &Request{
			Message: &wrp.Message{
				Destination: "mac:112233445566",
			},
		}

		connectionFactory = new(mockConnectionFactory)
		manager           = NewManager(nil, connectionFactory)
	)

	response, err := manager.Route(request)
	assert.Nil(response)
	assert.Equal(ErrorDeviceNotFound, err)

	connectionFactory.AssertExpectations(t)
}

func testManagerRouteNonUniqueID(t *testing.T) {
	var (
		assert  = assert.New(t)
		request = &Request{
			Message: &wrp.Message{
				Destination: "mac:112233445566",
			},
		}

		device1 = newDevice(ID("mac:112233445566"), Key("123"), nil, "", 1)
		device2 = newDevice(ID("mac:112233445566"), Key("234"), nil, "", 1)

		connectionFactory = new(mockConnectionFactory)
		manager           = NewManager(nil, connectionFactory).(*manager)
	)

	manager.registry.add(device1)
	manager.registry.add(device2)

	response, err := manager.Route(request)
	assert.Nil(response)
	assert.Equal(ErrorNonUniqueID, err)

	connectionFactory.AssertExpectations(t)
}

func testManagerPingPong(t *testing.T) {
	var (
		assert      = assert.New(t)
		connectWait = new(sync.WaitGroup)
		pongs       = make(chan Interface, 100)

		options = &Options{
			Logger: logging.TestLogger(t),
			Listeners: []Listener{
				func(event *Event) {
					switch event.Type {
					case Connect:
						connectWait.Done()
					case Pong:
						pongs <- event.Device
					}
				},
			},
			PingPeriod: 500 * time.Millisecond,
		}
	)

	connectWait.Add(testConnectionCount)

	var (
		_, server, connectURL = startWebsocketServer(options)
		dialer                = NewDialer(options, nil)
		testDevices           = connectTestDevices(t, assert, dialer, connectURL)
	)

	defer server.Close()
	defer closeTestDevices(assert, testDevices)
	connectWait.Wait()

	for id, connections := range testDevices {
		for _, c := range connections {
			// pongs are processed on the read goroutine
			go func(id ID, c Connection) {
				var err error
				for err == nil {
					_, err = c.NextReader()
				}
			}(id, c)
		}
	}

	pongWait := new(sync.WaitGroup)
	pongWait.Add(1)
	go func() {
		defer pongWait.Done()
		pongedDevices := make(deviceSet)
		timeout := time.After(10 * time.Second)
		for pongedDevices.len() < testConnectionCount {
			select {
			case ponged := <-pongs:
				pongedDevices.add(ponged)
			case <-timeout:
				assert.Fail("Not all devices responded to pings within the timeout")
			}
		}
	}()

	pongWait.Wait()
}

func TestManager(t *testing.T) {
	t.Run("Connect", func(t *testing.T) {
		t.Run("MissingDeviceContext", testManagerConnectMissingDeviceContext)
		t.Run("BadConveyHeader", testManagerConnectBadConveyHeader)
		t.Run("KeyError", testManagerConnectKeyError)
		t.Run("ConnectionFactoryError", testManagerConnectConnectionFactoryError)
		t.Run("Visit", testManagerConnectVisit)
	})

	t.Run("Route", func(t *testing.T) {
		t.Run("BadDestination", testManagerRouteBadDestination)
		t.Run("DeviceNotFound", testManagerRouteDeviceNotFound)
		t.Run("NonUniqueID", testManagerRouteNonUniqueID)
	})

	t.Run("Disconnect", testManagerDisconnect)
	t.Run("DisconnectOne", testManagerDisconnectOne)
	t.Run("DisconnectIf", testManagerDisconnectIf)

	t.Run("PongCallbackFor", testManagerPongCallbackFor)
	t.Run("PingPong", testManagerPingPong)
}

package device

import (
	"errors"
	"fmt"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/wrp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			&ConnectHandler{
				Logger:    o.logger(),
				Connector: manager,
			},
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
	assert.Error(err)
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
	assert.Error(err)
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
	assert.Error(err)
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
	assert.Error(err)
	assert.Equal(response.Code, http.StatusBadRequest)
}

func TestManagerPongCallbackFor(t *testing.T) {
	assert := assert.New(t)
	expectedDevice := newDevice(ID("ponged device"), Key("expected"), nil, 1)
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

func TestManagerConnectAndVisit(t *testing.T) {
	assert := assert.New(t)
	connectWait := new(sync.WaitGroup)
	connectWait.Add(testConnectionCount)
	connections := make(chan Interface, testConnectionCount)

	options := &Options{
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

func TestManagerDisconnectOne(t *testing.T) {
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

func TestManagerDisconnectIf(t *testing.T) {
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

func TestManagerRoute(t *testing.T) {
	var (
		assert      = assert.New(t)
		require     = require.New(t)
		connectWait = new(sync.WaitGroup)
		receiveWait = new(sync.WaitGroup)

		options = &Options{
			Logger: logging.TestLogger(t),
			Listeners: []Listener{
				func(event *Event) {
					switch event.Type {
					case Connect:
						connectWait.Done()
					case Disconnect:
						assert.True(event.Device.Closed())
						assert.Error(event.Device.Send(new(wrp.Message), nil))
					}
				},
			},
		}
	)

	connectWait.Add(testConnectionCount)
	receiveWait.Add(testConnectionCount)

	var (
		manager, server, connectURL = startWebsocketServer(options)
		dialer                      = NewDialer(options, nil)
		testDevices                 = connectTestDevices(t, assert, dialer, connectURL)
	)

	defer server.Close()
	defer closeTestDevices(assert, testDevices)

	for id, connections := range testDevices {
		for _, c := range connections {
			go func(id ID, c Connection) {
				defer receiveWait.Done()
				var (
					frame, err = c.NextReader()
					decoder    = wrp.NewDecoder(frame, wrp.Msgpack)
					message    wrp.Message
				)

				require.NotNil(frame)
				require.NoError(err)
				require.NoError(decoder.Decode(&message))
				assert.Equal(fmt.Sprintf("message for %s", id), string(message.Payload))
			}(id, c)
		}
	}

	connectWait.Wait()
	for id, expectedCount := range testDeviceIDs {
		// spawn a goroutine for each send to better detect any race conditions
		// or other concurrency issues
		go func(id ID, expectedCount int) {
			actualID, actualCount, err := manager.Route(
				&wrp.SimpleEvent{
					Destination: string(id),
					Payload:     []byte(fmt.Sprintf("message for %s", id)),
				},
				nil,
				func(d Interface, err error) {
					assert.Fail("The callback should not have been called")
				},
			)

			assert.Equal(id, actualID)
			assert.NoError(err)
			assert.Equal(expectedCount, actualCount)
		}(id, expectedCount)
	}

	receiveWait.Wait()

	id, count, err := manager.Route(
		&wrp.SimpleEvent{
			Destination: "nosuch device",
			Payload:     []byte("this shouldn't go anywhere"),
		},
		nil,
		func(Interface, error) {
			assert.Fail("The callback should not have been called")
		},
	)

	assert.Equal(ID(""), id)
	assert.Zero(count)
	assert.Error(err)
}

func TestManagerPingPong(t *testing.T) {
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

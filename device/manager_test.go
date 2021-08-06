package device

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"

	"github.com/xmidt-org/webpa-common/v2/convey"
	"github.com/xmidt-org/webpa-common/v2/xmetrics"

	"github.com/justinas/alice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/webpa-common/v2/logging"
	"github.com/xmidt-org/wrp-go/v3"
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
			alice.New(UseID.FromHeader).Then(
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

func testManagerConnectFilterDeny(t *testing.T) {
	assert := assert.New(t)
	mockFilter := new(mockFilter)
	options := &Options{
		Logger: log.NewNopLogger(),
		Filter: mockFilter,
	}

	manager := NewManager(options)
	response := httptest.NewRecorder()
	request := WithIDRequest(ID("mac:123412341234"), httptest.NewRequest("POST", "http://localhost.com", nil))

	mockFilter.On("AllowConnection", mock.Anything).Return(false, MatchResult{}).Once()

	device, err := manager.Connect(response, request, nil)
	assert.Nil(device)
	assert.Equal(err, ErrorDeviceFilteredOut)

}

func testManagerConnectMissingDeviceContext(t *testing.T) {
	assert := assert.New(t)
	options := &Options{
		Logger: log.NewNopLogger(),
	}

	manager := NewManager(options)
	response := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "http://localhost.com", nil)

	device, err := manager.Connect(response, request, nil)
	assert.Nil(device)
	assert.Error(err)
	assert.Equal(response.Code, http.StatusInternalServerError)
}

func testManagerConnectUpgradeError(t *testing.T) {
	var (
		assert  = assert.New(t)
		options = &Options{
			Logger: log.NewNopLogger(),
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
	)

	device, actualError := manager.Connect(response, request, responseHeader)
	assert.Nil(device)
	assert.Error(actualError)
}

func testManagerConnectVisit(t *testing.T) {
	var (
		assert      = assert.New(t)
		connectWait = new(sync.WaitGroup)
		connections = make(chan Interface, len(testDeviceIDs))

		options = &Options{
			Logger: log.NewNopLogger(),
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
	assert.Zero(manager.Disconnect(ID("nosuch"), CloseReason{}))
	for _, id := range testDeviceIDs {
		assert.Equal(true, manager.Disconnect(id, CloseReason{}))
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

	assert.Zero(manager.DisconnectIf(func(ID) (CloseReason, bool) { return CloseReason{}, false }))
	select {
	case <-disconnections:
		assert.Fail("No disconnections should have occurred")
	default:
		// the passing case
	}

	for _, id := range testDeviceIDs {
		assert.Equal(1, manager.DisconnectIf(func(candidate ID) (CloseReason, bool) { return CloseReason{}, candidate == id }))
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

func testManagerConnectIncludesConvey(t *testing.T) {
	var (
		assert      = assert.New(t)
		require     = require.New(t)
		connectWait = new(sync.WaitGroup)
		contents    = make(chan []byte, 1)

		options = &Options{
			Logger: log.NewNopLogger(),
			Listeners: []Listener{
				func(event *Event) {
					if event.Type == Connect {
						defer connectWait.Done()
						select {
						case contents <- event.Contents:
						default:
							assert.Fail("The connect listener should not block")
						}
					}
				},
			},
		}

		_, server, connectURL = startWebsocketServer(options)
	)

	defer server.Close()
	connectWait.Add(1)

	dialer := DefaultDialer()

	/*
		Convey header in base 64:
			{
				"hw-serial-number":123456789,
				"webpa-protocol":"WebPA-1.6"
			}

	*/
	header := &http.Header{
		"X-Webpa-Convey": {"eyAgDQogICAiaHctc2VyaWFsLW51bWJlciI6MTIzNDU2Nzg5LA0KICAgIndlYnBhLXByb3RvY29sIjoiV2ViUEEtMS42Ig0KfQ=="},
	}

	deviceConnection, _, err := dialer.DialDevice(string(testDeviceIDs[0]), connectURL, *header)
	require.NotNil(deviceConnection)
	require.NoError(err)

	defer assert.NoError(deviceConnection.Close())

	connectWait.Wait()
	close(contents)
	assert.Equal(1, len(contents))

	content := <-contents
	convey := make(map[string]interface{})
	err = json.Unmarshal(content, &convey)

	assert.Nil(err)
	assert.Equal(2, len(convey))
	assert.Equal(float64(123456789), convey["hw-serial-number"])
	assert.Equal("WebPA-1.6", convey["webpa-protocol"])
}

func TestManager(t *testing.T) {
	t.Run("Connect", func(t *testing.T) {
		t.Run("MissingDeviceContext", testManagerConnectMissingDeviceContext)
		t.Run("FilterOutDevice", testManagerConnectFilterDeny)
		t.Run("UpgradeError", testManagerConnectUpgradeError)
		t.Run("Visit", testManagerConnectVisit)
		t.Run("IncludesConvey", testManagerConnectIncludesConvey)
	})

	t.Run("Route", func(t *testing.T) {
		t.Run("BadDestination", testManagerRouteBadDestination)
		t.Run("DeviceNotFound", testManagerRouteDeviceNotFound)
	})

	t.Run("Disconnect", testManagerDisconnect)
	t.Run("DisconnectIf", testManagerDisconnectIf)
}

func TestGaugeCardinality(t *testing.T) {
	var (
		assert = assert.New(t)
		r, err = xmetrics.NewRegistry(nil, Metrics)
		m      = NewManager(&Options{
			MetricsProvider: r,
		})
	)
	assert.NoError(err)

	assert.NotPanics(func() {
		dec, err := m.(*manager).conveyHWMetric.Update(convey.C{"hw-model": "cardinality", "fw-name": "firmware-number", "model": "f"}, "partnerid", "comcast", "trust", "0")
		assert.NoError(err)
		dec()
	})

	assert.Panics(func() {
		m.(*manager).measures.Models.With("neat", "bad").Add(-1)
	})
}

func TestWRPSourceIsValid(t *testing.T) {
	assert := assert.New(t)
	canonicalID := ID("mac:112233445566")
	testData := []struct {
		Name           string
		Source         string
		IsValid        bool
		BaseLabelPairs map[string]string
	}{
		{
			Name:    "EmptySource",
			IsValid: false,
			Source: "   	",
			BaseLabelPairs: map[string]string{"reason": "empty"},
		},

		{
			Name:           "ParseFailure",
			IsValid:        false,
			Source:         "serial>hacker/service",
			BaseLabelPairs: map[string]string{"reason": "parse_error"},
		},
		{
			Name:           "IDMismatch",
			IsValid:        false,
			Source:         "mac:665544332211/service/some/path",
			BaseLabelPairs: map[string]string{"reason": "id_mismatch"},
		},
		{
			Name:           "IDMatch",
			IsValid:        true,
			Source:         "mac:112233445566/service/some/path",
			BaseLabelPairs: map[string]string{"reason": "id_match"},
		},
	}

	for _, record := range testData {
		t.Run(record.Name, func(t *testing.T) {
			expectedStrictLabels, expectedLenientLabels := createLabelMaps(!record.IsValid, record.BaseLabelPairs)

			d := new(device)
			d.id = canonicalID
			d.errorLog = log.WithPrefix(logging.NewTestLogger(nil, t), "id", canonicalID)
			d.metadata = new(Metadata)

			// strict mode
			counter := newTestCounter()
			message := &wrp.Message{Source: record.Source}
			m := &manager{enforceWRPSourceCheck: true, measures: Measures{WRPSourceCheck: counter}}
			ok := m.wrpSourceIsValid(message, d)
			assert.Equal(record.IsValid, ok)
			assert.Equal(expectedStrictLabels, counter.labelPairs)

			// lenient mode
			counter = newTestCounter()
			message = &wrp.Message{Source: record.Source}
			m = &manager{enforceWRPSourceCheck: false, measures: Measures{WRPSourceCheck: counter}}

			ok = m.wrpSourceIsValid(message, d)
			assert.True(ok)
			assert.Equal(expectedLenientLabels, counter.labelPairs)
		})
	}

}

func createLabelMaps(rejected bool, baseLabelPairs map[string]string) (strict map[string]string, lenient map[string]string) {
	strict = make(map[string]string)
	lenient = make(map[string]string)

	for k, v := range baseLabelPairs {
		strict[k] = v
		lenient[k] = v
	}

	if rejected {
		strict["outcome"] = "rejected"
	} else {
		strict["outcome"] = "accepted"
	}
	lenient["outcome"] = "accepted"

	return
}

type testCounter struct {
	count      float64
	labelPairs map[string]string
}

func (c *testCounter) Add(delta float64) {
	c.count += delta
}

func (c *testCounter) With(labelValues ...string) metrics.Counter {
	for i := 0; i < len(labelValues)-1; i += 2 {
		c.labelPairs[labelValues[i]] = labelValues[i+1]
	}
	return c
}

func newTestCounter() *testCounter {
	return &testCounter{
		labelPairs: make(map[string]string),
	}
}

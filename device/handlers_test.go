package device

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/wrp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func testMessageHandlerLogger(t *testing.T) {
	var (
		assert = assert.New(t)
		logger = logging.TestLogger(t)

		handler = MessageHandler{}
	)

	assert.NotNil(handler.logger())

	handler.Logger = logger
	assert.Equal(logger, handler.logger())
}

func testMessageHandlerCreateContextNoTimeout(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		handler = MessageHandler{}
	)

	ctx, cancel := handler.createContext(httptest.NewRequest("GET", "/", nil))
	require.NotNil(ctx)
	require.NotNil(cancel)

	deadline, ok := ctx.Deadline()
	assert.WithinDuration(time.Now(), deadline, DefaultMessageTimeout)
	assert.True(ok)

	cancel()
	select {
	case <-ctx.Done():
		// passing
	default:
		assert.Fail("The cancel function should have cancelled the context")
	}
}

func testMessageHandlerCreateContextWithTimeout(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		handler = MessageHandler{
			Timeout: 247 * time.Hour,
		}
	)

	ctx, cancel := handler.createContext(httptest.NewRequest("GET", "/", nil))
	require.NotNil(ctx)
	require.NotNil(cancel)

	deadline, ok := ctx.Deadline()
	assert.WithinDuration(time.Now(), deadline, handler.Timeout)
	assert.True(ok)

	cancel()
	select {
	case <-ctx.Done():
		// passing
	default:
		assert.Fail("The cancel function should have cancelled the context")
	}
}

func testMessageHandlerServeHTTPDecodeError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		invalidContents    = []byte("this is not a valid WRP message")
		response           = httptest.NewRecorder()
		request            = httptest.NewRequest("GET", "/foo", bytes.NewReader(invalidContents))
		actualResponseBody map[string]interface{}

		router  = new(mockRouter)
		handler = MessageHandler{
			Decoders: wrp.NewDecoderPool(1, wrp.Msgpack),
			Router:   router,
		}
	)

	handler.ServeHTTP(response, request)
	assert.Equal(http.StatusBadRequest, response.Code)
	assert.Equal("application/json", response.HeaderMap.Get("Content-Type"))
	responseContents, err := ioutil.ReadAll(response.Body)
	require.NoError(err)
	assert.NoError(json.Unmarshal(responseContents, &actualResponseBody))

	router.AssertExpectations(t)
}

func testMessageHandlerServeHTTPRouteError(t *testing.T, routeError error, expectedCode int) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		message = &wrp.Message{
			Type:        wrp.SimpleEventMessageType,
			Source:      "test.com",
			Destination: "mac:123412341234",
		}

		setupEncoders   = wrp.NewEncoderPool(1, wrp.Msgpack)
		requestContents []byte
	)

	require.NoError(setupEncoders.EncodeBytes(&requestContents, message))

	var (
		response           = httptest.NewRecorder()
		request            = httptest.NewRequest("POST", "/foo", bytes.NewReader(requestContents))
		actualResponseBody map[string]interface{}

		router  = new(mockRouter)
		handler = MessageHandler{
			Router:   router,
			Decoders: wrp.NewDecoderPool(1, wrp.Msgpack),
		}
	)

	router.On(
		"Route",
		mock.MatchedBy(func(candidate *Request) bool {
			return candidate.Message != nil &&
				len(candidate.Contents) > 0 &&
				candidate.Format == wrp.Msgpack
		}),
	).Once().Return(nil, routeError)

	handler.ServeHTTP(response, request)
	assert.Equal(expectedCode, response.Code)
	assert.Equal("application/json", response.HeaderMap.Get("Content-Type"))
	responseContents, err := ioutil.ReadAll(response.Body)
	require.NoError(err)
	assert.NoError(json.Unmarshal(responseContents, &actualResponseBody))

	router.AssertExpectations(t)
}

func testMessageHandlerServeHTTPEvent(t *testing.T, requestFormat wrp.Format) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		event = &wrp.SimpleEvent{
			Source:      "test.com",
			Destination: "mac:123412341234",
			ContentType: "text/plain",
			Payload:     []byte("some lovely data here"),
			Headers:     []string{"Header-1", "Header-2"},
			Metadata:    map[string]string{"foo": "bar"},
		}

		setupEncoders   = wrp.NewEncoderPool(1, requestFormat)
		requestContents []byte
	)

	require.NoError(setupEncoders.EncodeBytes(&requestContents, event))

	var (
		response = httptest.NewRecorder()
		request  = httptest.NewRequest("POST", "/foo", bytes.NewReader(requestContents))

		router  = new(mockRouter)
		handler = MessageHandler{
			Router:   router,
			Decoders: wrp.NewDecoderPool(1, requestFormat),
		}

		actualDeviceRequest *Request
	)

	router.On(
		"Route",
		mock.MatchedBy(func(candidate *Request) bool {
			actualDeviceRequest = candidate
			return candidate.Message != nil &&
				len(candidate.Contents) > 0 &&
				candidate.Format == requestFormat
		}),
	).Once().Return(nil, nil)

	handler.ServeHTTP(response, request)
	assert.Equal(http.StatusOK, response.Code)
	assert.Equal(0, response.Body.Len())
	require.NotNil(actualDeviceRequest)

	router.AssertExpectations(t)
}

func testMessageHandlerServeHTTPRequestResponse(t *testing.T, responseFormat, requestFormat wrp.Format) {
	const (
		transactionKey = "transaction-key"
		encodedConvey  = "expected encoded convey"
	)

	var (
		assert  = assert.New(t)
		require = require.New(t)

		requestMessage = &wrp.Message{
			Type:            wrp.SimpleRequestResponseMessageType,
			Source:          "test.com",
			Destination:     "mac:123412341234",
			TransactionUUID: transactionKey,
			ContentType:     "text/plain",
			Payload:         []byte("some lovely data here"),
			Headers:         []string{"Header-1", "Header-2"},
			Metadata:        map[string]string{"foo": "bar"},
		}

		responseMessage = &wrp.Message{
			Type:            wrp.SimpleRequestResponseMessageType,
			Destination:     "test.com",
			Source:          "mac:123412341234",
			TransactionUUID: transactionKey,
		}

		setupRequestEncoders   = wrp.NewEncoderPool(1, requestFormat)
		setupResponseEncoders  = wrp.NewEncoderPool(1, wrp.Msgpack)
		verifyResponseDecoders = wrp.NewDecoderPool(1, responseFormat)
		requestContents        []byte
		responseContents       []byte
	)

	require.NoError(setupRequestEncoders.EncodeBytes(&requestContents, requestMessage))
	require.NoError(setupResponseEncoders.EncodeBytes(&responseContents, responseMessage))

	var (
		response = httptest.NewRecorder()
		request  = httptest.NewRequest("POST", "/foo", bytes.NewReader(requestContents))

		router  = new(mockRouter)
		device  = new(mockDevice)
		handler = MessageHandler{
			Router:   router,
			Decoders: wrp.NewDecoderPool(1, requestFormat),
			Encoders: wrp.NewEncoderPool(1, responseFormat),
		}

		actualDeviceRequest    *Request
		expectedDeviceResponse = &Response{
			Device:   device,
			Message:  responseMessage,
			Format:   wrp.Msgpack,
			Contents: responseContents,
		}
	)

	device.On("EncodedConvey").Once().Return(encodedConvey)

	router.On(
		"Route",
		mock.MatchedBy(func(candidate *Request) bool {
			actualDeviceRequest = candidate
			return candidate.Message != nil &&
				len(candidate.Contents) > 0 &&
				candidate.Format == requestFormat
		}),
	).Once().Return(expectedDeviceResponse, nil)

	handler.ServeHTTP(response, request)
	assert.Equal(http.StatusOK, response.Code)
	assert.Equal(responseFormat.ContentType(), response.HeaderMap.Get("Content-Type"))
	assert.Equal(encodedConvey, response.HeaderMap.Get(ConveyHeader))
	require.NotNil(actualDeviceRequest)
	assert.NoError(verifyResponseDecoders.Decode(new(wrp.Message), response.Body))

	router.AssertExpectations(t)
	device.AssertExpectations(t)
}

func testMessageHandlerServeHTTPEncodeError(t *testing.T) {
	const (
		transactionKey = "transaction-key"
		encodedConvey  = "expected encoded convey"
	)

	var (
		assert  = assert.New(t)
		require = require.New(t)

		requestMessage = &wrp.Message{
			Type:            wrp.SimpleRequestResponseMessageType,
			Source:          "test.com",
			Destination:     "mac:123412341234",
			TransactionUUID: transactionKey,
			ContentType:     "text/plain",
			Payload:         []byte("some lovely data here"),
			Headers:         []string{"Header-1", "Header-2"},
			Metadata:        map[string]string{"foo": "bar"},
		}

		responseMessage = &wrp.Message{
			Type:            wrp.SimpleRequestResponseMessageType,
			Destination:     "test.com",
			Source:          "mac:123412341234",
			TransactionUUID: transactionKey,
		}

		setupRequestEncoders = wrp.NewEncoderPool(1, wrp.Msgpack)
		requestContents      []byte
	)

	require.NoError(setupRequestEncoders.EncodeBytes(&requestContents, requestMessage))

	var (
		response = httptest.NewRecorder()
		request  = httptest.NewRequest("POST", "/foo", bytes.NewReader(requestContents))

		router  = new(mockRouter)
		device  = new(mockDevice)
		handler = MessageHandler{
			Router:   router,
			Decoders: wrp.NewDecoderPool(1, wrp.Msgpack),
		}

		actualResponseBody     map[string]interface{}
		expectedDeviceResponse = &Response{
			Device:  device,
			Message: responseMessage,
			Format:  wrp.Msgpack,
		}
	)

	device.On("EncodedConvey").Once().Return(encodedConvey)

	router.On(
		"Route",
		mock.MatchedBy(func(candidate *Request) bool {
			return candidate.Message != nil &&
				len(candidate.Contents) > 0 &&
				candidate.Format == wrp.Msgpack
		}),
	).Once().Return(expectedDeviceResponse, nil)

	handler.ServeHTTP(response, request)
	assert.Equal(http.StatusInternalServerError, response.Code)
	assert.Equal(encodedConvey, response.HeaderMap.Get(ConveyHeader))
	assert.Equal("application/json", response.HeaderMap.Get("Content-Type"))
	responseContents, err := ioutil.ReadAll(response.Body)
	require.NoError(err)
	assert.NoError(json.Unmarshal(responseContents, &actualResponseBody))

	router.AssertExpectations(t)
	device.AssertExpectations(t)
}

func TestMessageHandler(t *testing.T) {
	t.Run("Logger", testMessageHandlerLogger)
	t.Run("CreateContext", func(t *testing.T) {
		t.Run("NoTimeout", testMessageHandlerCreateContextNoTimeout)
		t.Run("WithTimeout", testMessageHandlerCreateContextWithTimeout)
	})

	t.Run("ServeHTTP", func(t *testing.T) {
		t.Run("DecodeError", testMessageHandlerServeHTTPDecodeError)
		t.Run("EncodeError", testMessageHandlerServeHTTPEncodeError)

		t.Run("RouteError", func(t *testing.T) {
			testMessageHandlerServeHTTPRouteError(t, ErrorInvalidDeviceName, http.StatusBadRequest)
			testMessageHandlerServeHTTPRouteError(t, ErrorDeviceNotFound, http.StatusNotFound)
			testMessageHandlerServeHTTPRouteError(t, ErrorNonUniqueID, http.StatusBadRequest)
			testMessageHandlerServeHTTPRouteError(t, ErrorInvalidTransactionKey, http.StatusBadRequest)
			testMessageHandlerServeHTTPRouteError(t, ErrorTransactionAlreadyRegistered, http.StatusBadRequest)
			testMessageHandlerServeHTTPRouteError(t, errors.New("random error"), http.StatusInternalServerError)
		})

		t.Run("Event", func(t *testing.T) {
			for _, requestFormat := range []wrp.Format{wrp.Msgpack, wrp.JSON} {
				testMessageHandlerServeHTTPEvent(t, requestFormat)
			}
		})

		t.Run("RequestResponse", func(t *testing.T) {
			for _, responseFormat := range []wrp.Format{wrp.Msgpack, wrp.JSON} {
				for _, requestFormat := range []wrp.Format{wrp.Msgpack, wrp.JSON} {
					testMessageHandlerServeHTTPRequestResponse(t, responseFormat, requestFormat)
				}
			}
		})
	})
}

func testConnectHandlerLogger(t *testing.T) {
	var (
		assert = assert.New(t)
		logger = logging.TestLogger(t)

		handler = ConnectHandler{}
	)

	assert.NotNil(handler.logger())

	handler.Logger = logger
	assert.Equal(logger, handler.logger())
}

func testConnectHandlerServeHTTP(t *testing.T, connectError error, responseHeader http.Header) {
	var (
		assert = assert.New(t)

		device    = new(mockDevice)
		connector = new(mockConnector)
		handler   = ConnectHandler{
			Connector:      connector,
			ResponseHeader: responseHeader,
		}

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "/", nil)
	)

	if connectError != nil {
		connector.On("Connect", response, request, responseHeader).Once().Return(nil, connectError)
	} else {
		device.On("ID").Once().Return(ID("mac:112233445566"))
		connector.On("Connect", response, request, responseHeader).Once().Return(device, connectError)
	}

	handler.ServeHTTP(response, request)

	// the handler itself shouldn't do anything to the response.
	// the Connector does that
	assert.Equal(http.StatusOK, response.Code)

	device.AssertExpectations(t)
	connector.AssertExpectations(t)
}

func TestConnectHandler(t *testing.T) {
	t.Run("Logger", testConnectHandlerLogger)
	t.Run("ServeHTTP", func(t *testing.T) {
		testConnectHandlerServeHTTP(t, nil, nil)
		testConnectHandlerServeHTTP(t, nil, http.Header{"Header-1": []string{"Value-1"}})
		testConnectHandlerServeHTTP(t, errors.New("expected error"), nil)
		testConnectHandlerServeHTTP(t, errors.New("expected error"), http.Header{"Header-1": []string{"Value-1"}})
	})
}

func testConnectedDeviceListenerRefreshInterval(t *testing.T) {
	var (
		assert   = assert.New(t)
		listener = new(ConnectedDeviceListener)
	)

	assert.Equal(DefaultRefreshInterval, listener.refreshInterval())

	listener.RefreshInterval = 72 * time.Minute
	assert.Equal(listener.RefreshInterval, listener.refreshInterval())
}

func testConnectedDeviceListenerNewTick(t *testing.T) {
	var (
		assert   = assert.New(t)
		listener = new(ConnectedDeviceListener)
	)

	tickerC, stop := listener.newTick()
	assert.NotNil(tickerC)
	assert.NotNil(stop)
	stop()

	var (
		tickTime         = time.Now()
		customC          = make(chan time.Time, 1)
		customStopCalled bool
		customStop       = func() { customStopCalled = true }
	)

	listener.Tick = func(time.Duration) (<-chan time.Time, func()) {
		return customC, customStop
	}

	tickerC, stop = listener.newTick()
	assert.NotNil(tickerC)
	customC <- tickTime
	assert.Equal(tickTime, <-tickerC)

	assert.NotNil(stop)
	stop()
	assert.True(customStopCalled)
}

func expectNoDeviceListUpdate(assert *assert.Assertions, updates <-chan []byte) {
	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()
	select {
	case update := <-updates:
		// this allwos the channel to have been closed
		assert.True(update == nil, "No updates should have happened")
	case <-timer.C:
		// passing
	}
}

func expectDeviceListUpdate(assert *assert.Assertions, updates <-chan []byte, expectedKeys ...string) {
	var (
		timer   = time.NewTimer(100 * time.Millisecond)
		rawJSON []byte
		update  map[string]interface{}
	)

	defer timer.Stop()
	select {
	case rawJSON = <-updates:
	case <-timer.C:
		assert.Fail("An update should have occurred")
		return
	}

	if !assert.NoError(json.Unmarshal(rawJSON, &update)) {
		return
	}

	actualDevices := update["devices"].([]interface{})
	if !assert.Equal(len(actualDevices), len(expectedKeys)) {
		return
	}

	for _, rawDevice := range actualDevices {
		actualDevice := rawDevice.(map[string]interface{})
		actualKey := actualDevice["key"].(string)
		found := false
		for i := 0; i < len(expectedKeys) && !found; i++ {
			found = (actualKey == expectedKeys[i])
		}

		assert.True(found, "%s was not an expected device key in %v", actualKey, expectedKeys)
	}
}

func testConnectedDeviceListenerListen(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		deviceA = new(mockDevice)
		deviceB = new(mockDevice)

		refreshC    = make(chan time.Time, 1)
		refreshStop = func() { close(refreshC) }
		listener    = ConnectedDeviceListener{
			Tick: func(time.Duration) (<-chan time.Time, func()) { return refreshC, refreshStop },
		}

		deviceListener, updates = listener.Listen()
	)

	require.NotNil(deviceListener)
	require.NotNil(updates)

	deviceA.On("Key").Return(Key("A"))
	deviceA.On("String").Return(`{"id": "A", "key": "A"}`)
	deviceB.On("Key").Return(Key("B"))
	deviceB.On("String").Return(`{"id": "B", "key": "B"}`)

	refreshC <- time.Now()
	expectNoDeviceListUpdate(assert, updates)

	deviceListener(&Event{Type: Connect, Device: deviceA})
	deviceListener(&Event{Type: Pong, Device: deviceB})
	refreshC <- time.Now()
	expectDeviceListUpdate(assert, updates, "A")

	deviceListener(&Event{Type: Disconnect, Device: deviceA})
	deviceListener(&Event{Type: Connect, Device: deviceA})
	refreshC <- time.Now()
	expectDeviceListUpdate(assert, updates, "A")

	deviceListener(&Event{Type: Connect, Device: deviceB})
	refreshC <- time.Now()
	expectDeviceListUpdate(assert, updates, "A", "B")

	deviceListener(&Event{Type: TransactionComplete})
	deviceListener(&Event{Type: MessageFailed})
	refreshC <- time.Now()
	expectNoDeviceListUpdate(assert, updates)

	deviceListener(&Event{Type: Disconnect, Device: deviceA})
	refreshC <- time.Now()
	expectDeviceListUpdate(assert, updates, "B")

	deviceListener(&Event{Type: Disconnect, Device: deviceB})
	refreshC <- time.Now()
	expectDeviceListUpdate(assert, updates)

	listener.Stop()
	deviceListener(&Event{Type: Connect, Device: deviceB})
	expectNoDeviceListUpdate(assert, updates)
}

func TestConnectedDeviceListener(t *testing.T) {
	t.Run("RefreshInterval", testConnectedDeviceListenerRefreshInterval)
	t.Run("NewTick", testConnectedDeviceListenerNewTick)
	t.Run("Listen", testConnectedDeviceListenerListen)
}

func testListHandlerServeHTTPUninitialized(t *testing.T) {
	var (
		assert   = assert.New(t)
		handler  = new(ListHandler)
		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "/", nil)
	)

	handler.ServeHTTP(response, request)
	assert.Equal(http.StatusServiceUnavailable, response.Code)
}

func expectDeviceListResponse(assert *assert.Assertions, handler http.Handler, expectedJSON string) {
	var (
		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "/", nil)
	)

	handler.ServeHTTP(response, request)
	assert.Equal(http.StatusOK, response.Code)
	assert.Equal("application/json", response.HeaderMap.Get("Content-Type"))

	rawJSON, err := ioutil.ReadAll(response.Body)
	if !assert.NoError(err) {
		return
	}

	assert.JSONEq(expectedJSON, string(rawJSON))
}

func testListHandlerServeHTTPWhileConsuming(t *testing.T) {
	var (
		assert  = assert.New(t)
		updates = make(chan []byte, 1)
		handler = new(ListHandler)
	)

	handler.Consume(updates)
	expectDeviceListResponse(assert, handler, `{"devices":[]}`)

	old := handler.cachedJSON.Load().([]byte)
	updates <- []byte(`{"devices":[{"id": "A"},{"id": "B"}]}`)
	for bytes.Equal(old, handler.cachedJSON.Load().([]byte)) {
		// spin until we get a change
		time.Sleep(100 * time.Millisecond)
	}
	expectDeviceListResponse(assert, handler, `{"devices":[{"id": "A"},{"id": "B"}]}`)

	close(updates)
	time.Sleep(100 * time.Millisecond)
	expectDeviceListResponse(assert, handler, `{"devices":[{"id": "A"},{"id": "B"}]}`)

	// start consuming again ...
	updates = make(chan []byte, 1)
	handler.Consume(updates)
	time.Sleep(100 * time.Millisecond)
	expectDeviceListResponse(assert, handler, `{"devices":[{"id": "A"},{"id": "B"}]}`)

	old = handler.cachedJSON.Load().([]byte)
	updates <- []byte(`{"devices":[{"id": "C"},{"id": "D"},{"id": "E"}]}`)
	for bytes.Equal(old, handler.cachedJSON.Load().([]byte)) {
		// spin until we get a change
		time.Sleep(100 * time.Millisecond)
	}
	expectDeviceListResponse(assert, handler, `{"devices":[{"id": "C"},{"id": "D"},{"id": "E"}]}`)

	close(updates)
	time.Sleep(100 * time.Millisecond)
	expectDeviceListResponse(assert, handler, `{"devices":[{"id": "C"},{"id": "D"},{"id": "E"}]}`)
}

func TestListHandler(t *testing.T) {
	t.Run("ServeHTTP", func(t *testing.T) {
		t.Run("Uninitialized", testListHandlerServeHTTPUninitialized)
		t.Run("WhileConsuming", testListHandlerServeHTTPWhileConsuming)
	})
}

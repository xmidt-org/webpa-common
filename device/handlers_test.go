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
	const transactionKey = "transaction-key"

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
		handler = MessageHandler{
			Router:   router,
			Decoders: wrp.NewDecoderPool(1, requestFormat),
			Encoders: wrp.NewEncoderPool(1, responseFormat),
		}

		actualDeviceRequest    *Request
		expectedDeviceResponse = &Response{
			Message:  responseMessage,
			Format:   wrp.Msgpack,
			Contents: responseContents,
		}
	)

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
	require.NotNil(actualDeviceRequest)
	assert.NoError(verifyResponseDecoders.Decode(new(wrp.Message), response.Body))

	router.AssertExpectations(t)
}

func testMessageHandlerServeHTTPEncodeError(t *testing.T) {
	const transactionKey = "transaction-key"

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
		handler = MessageHandler{
			Router:   router,
			Decoders: wrp.NewDecoderPool(1, wrp.Msgpack),
		}

		actualResponseBody     map[string]interface{}
		expectedDeviceResponse = &Response{
			Message: responseMessage,
			Format:  wrp.Msgpack,
		}
	)

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
	assert.Equal("application/json", response.HeaderMap.Get("Content-Type"))
	responseContents, err := ioutil.ReadAll(response.Body)
	require.NoError(err)
	assert.NoError(json.Unmarshal(responseContents, &actualResponseBody))

	router.AssertExpectations(t)
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

func testListHandlerRefreshInterval(t *testing.T) {
	var (
		assert  = assert.New(t)
		handler = ListHandler{}
	)

	assert.Equal(DefaultRefreshInterval, handler.refreshInterval())

	handler.RefreshInterval = 72 * time.Minute
	assert.Equal(handler.RefreshInterval, handler.refreshInterval())
}

func testListHandlerBacklog(t *testing.T) {
	var (
		assert  = assert.New(t)
		handler = ListHandler{}
	)

	assert.Equal(DefaultListBacklog, handler.backlog())

	handler.Backlog = 56792
	assert.Equal(handler.Backlog, handler.backlog())
}

func testListHandlerNewTick(t *testing.T) {
	var (
		assert  = assert.New(t)
		handler = ListHandler{}
	)

	tickerC, stop := handler.newTick()
	assert.NotNil(tickerC)
	assert.NotNil(stop)
	stop()

	var (
		tickTime         = time.Now()
		customC          = make(chan time.Time, 1)
		customStopCalled bool
		customStop       = func() { customStopCalled = true }
	)

	handler.Tick = func(time.Duration) (<-chan time.Time, func()) {
		return customC, customStop
	}

	tickerC, stop = handler.newTick()
	assert.NotNil(tickerC)
	customC <- tickTime
	assert.Equal(tickTime, <-tickerC)

	assert.NotNil(stop)
	stop()
	assert.True(customStopCalled)
}

// assertDeviceList asserts that the handler produces one of the supplied JSON documents.
// Because iteration over map keys is not consistent, this function allows several JSON alternatives
// for the expected JSON.
func assertDeviceList(assert *assert.Assertions, handler http.Handler, expectedJSON ...string) {
	var (
		request  = httptest.NewRequest("GET", "/", nil)
		response = httptest.NewRecorder()
	)

	handler.ServeHTTP(response, request)
	assert.Equal(http.StatusOK, response.Code)
	assert.Equal("application/json", response.HeaderMap.Get("Content-Type"))

	actualJSON, err := ioutil.ReadAll(response.Body)
	if !assert.NoError(err) {
		return
	}

	if len(expectedJSON) == 1 {
		assert.JSONEq(expectedJSON[0], string(actualJSON))
	} else if len(expectedJSON) > 1 {
		for _, candidate := range expectedJSON {
			// the best we can do here is an exact match
			if bytes.Equal(actualJSON, []byte(candidate)) {
				return
			}
		}

		assert.Fail("Invalid response body", "%s does not match any of %v", actualJSON, expectedJSON)
	}
}

func testListHandlerServeHTTPUninitialized(t *testing.T) {
	var (
		assert  = assert.New(t)
		handler = &ListHandler{}

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "/", nil)
	)

	handler.ServeHTTP(response, request)
	assert.Equal(response.Code, http.StatusServiceUnavailable)
}

func testListHandlerServeHTTPWithEvents(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		deviceA = new(mockDevice)
		deviceB = new(mockDevice)

		refreshC    = make(chan time.Time, 1)
		refreshStop = func() { close(refreshC) }
		handler     = &ListHandler{
			Tick: func(time.Duration) (<-chan time.Time, func()) {
				return refreshC, refreshStop
			},
		}

		listener = handler.Listen()
	)

	require.NotNil(listener)
	deviceA.On("Key").Return(Key("A"))
	deviceA.On("String").Return(`{"id": "A"}`)
	deviceB.On("Key").Return(Key("B"))
	deviceB.On("String").Return(`{"id": "B"}`)
	assertDeviceList(assert, handler, `{"devices":[]}`)

	listener(&Event{Type: Connect, Device: deviceA})

	// refresh hasn't run yet ...
	assertDeviceList(assert, handler, `{"devices":[]}`)

	old := handler.cachedJSON.Load().([]byte)
	refreshC <- time.Now()
	for bytes.Equal(old, handler.cachedJSON.Load().([]byte)) {
		// spin until we get a change
	}
	assertDeviceList(assert, handler, `{"devices":[{"id": "A"}]}`)

	listener(&Event{Type: Connect, Device: deviceB})
	old = handler.cachedJSON.Load().([]byte)
	refreshC <- time.Now()
	for bytes.Equal(old, handler.cachedJSON.Load().([]byte)) {
		// spin until we get a change
	}
	assertDeviceList(assert, handler, `{"devices":[{"id": "B"},{"id": "A"}]}`, `{"devices":[{"id": "A"},{"id": "B"}]}`)

	listener(&Event{Type: Disconnect, Device: deviceA})
	old = handler.cachedJSON.Load().([]byte)
	refreshC <- time.Now()
	for bytes.Equal(old, handler.cachedJSON.Load().([]byte)) {
		// spin until we get a change
	}
	assertDeviceList(assert, handler, `{"devices":[{"id": "B"}]}`)

	handler.Stop()

	handler.lock.Lock()
	assert.Nil(handler.shutdown)
	handler.lock.Unlock()
}

func TestListHandler(t *testing.T) {
	t.Run("RefreshInterval", testListHandlerRefreshInterval)
	t.Run("Backlog", testListHandlerBacklog)
	t.Run("NewTick", testListHandlerNewTick)
	t.Run("ServeHTTP", func(t *testing.T) {
		t.Run("Uninitialized", testListHandlerServeHTTPUninitialized)
		t.Run("WithEvents", testListHandlerServeHTTPWithEvents)
	})
}

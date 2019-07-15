package device

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/wrp"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func testTimeout(o *Options, t *testing.T) {
	var (
		assert         = assert.New(t)
		require        = require.New(t)
		request        = httptest.NewRequest("GET", "/", nil)
		response       = httptest.NewRecorder()
		ctx            context.Context
		delegateCalled bool

		handler = alice.New(Timeout(o)).Then(
			http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				delegateCalled = true
				ctx = request.Context()
				assert.NotEqual(context.Background(), ctx)

				deadline, ok := ctx.Deadline()
				assert.False(deadline.IsZero())
				assert.True(deadline.Sub(time.Now()) <= o.requestTimeout())
				assert.True(ok)
			}),
		)
	)

	handler.ServeHTTP(response, request)
	require.True(delegateCalled)

	select {
	case <-ctx.Done():
		// pass
	default:
		assert.Fail("The context should have been cancelled after ServeHTTP exits")
	}
}

func TestTimeout(t *testing.T) {
	t.Run(
		"NilOptions",
		func(t *testing.T) { testTimeout(nil, t) },
	)

	t.Run(
		"DefaultOptions",
		func(t *testing.T) { testTimeout(new(Options), t) },
	)

	t.Run(
		"CustomOptions",
		func(t *testing.T) { testTimeout(&Options{RequestTimeout: 17 * time.Second}, t) },
	)
}

func testUseIDFNilStrategy(t *testing.T) {
	var (
		assert   = assert.New(t)
		request  = httptest.NewRequest("GET", "/", nil)
		response = httptest.NewRecorder()

		handler = alice.New(useID(nil)).Then(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			assert.Fail("The delegate should not have been called")
		}))
	)

	assert.Panics(func() {
		handler.ServeHTTP(response, request)
	})
}

func testUseIDFError(t *testing.T) {
	var (
		assert         = assert.New(t)
		request        = httptest.NewRequest("GET", "/", nil)
		response       = httptest.NewRecorder()
		expectedError  = errors.New("expected")
		strategyCalled bool

		strategy = func(*http.Request) (ID, error) {
			strategyCalled = true
			return invalidID, expectedError
		}

		handler = alice.New(useID(strategy)).Then(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			assert.Fail("The delegate should not have been called")
		}))
	)

	handler.ServeHTTP(response, request)
	assert.True(strategyCalled)
}

func testUseIDFromHeaderMissing(t *testing.T) {
	var (
		assert   = assert.New(t)
		request  = httptest.NewRequest("GET", "/", nil)
		response = httptest.NewRecorder()

		handler = alice.New(UseID.FromHeader).Then(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			assert.Fail("The delegate should not have been called")
		}))
	)

	handler.ServeHTTP(response, request)
}

func testUseIDFromHeader(t *testing.T) {
	var (
		assert         = assert.New(t)
		require        = require.New(t)
		request        = httptest.NewRequest("GET", "/", nil)
		response       = httptest.NewRecorder()
		delegateCalled bool

		handler = alice.New(UseID.FromHeader).Then(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			delegateCalled = true
			id, ok := GetID(request.Context())
			assert.Equal(id, ID("mac:112233445566"))
			assert.True(ok)
		}))
	)

	request.Header.Set(DeviceNameHeader, "mac:112233445566")
	handler.ServeHTTP(response, request)
	require.True(delegateCalled)
}

func testUseIDFromPath(t *testing.T) {
	var (
		assert         = assert.New(t)
		request        = httptest.NewRequest("GET", "/test/mac:112233445566", nil)
		response       = httptest.NewRecorder()
		delegateCalled bool

		handler = alice.New(UseID.FromPath("did")).Then(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			delegateCalled = true
			id, ok := GetID(request.Context())
			assert.Equal(id, ID("mac:112233445566"))
			assert.True(ok)
		}))

		router = mux.NewRouter()
	)

	router.Handle("/test/{did}", handler)
	router.ServeHTTP(response, request)
	assert.Equal(http.StatusOK, response.Code)
	assert.True(delegateCalled)
}

func testUseIDFromPathMissingVars(t *testing.T) {
	var (
		assert   = assert.New(t)
		request  = httptest.NewRequest("GET", "/foo", nil)
		response = httptest.NewRecorder()

		handler = alice.New(UseID.FromPath("did")).Then(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			assert.Fail("The delegate should not have been called")
		}))
	)

	handler.ServeHTTP(response, request)
	assert.Equal(http.StatusBadRequest, response.Code)
}

func testUseIDFromPathMissingDeviceNameVar(t *testing.T) {
	var (
		assert   = assert.New(t)
		request  = httptest.NewRequest("GET", "/foo", nil)
		response = httptest.NewRecorder()

		handler = alice.New(UseID.FromPath("did")).Then(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			assert.Fail("The delegate should not have been called")
		}))

		router = mux.NewRouter()
	)

	router.Handle("/foo", handler)
	router.ServeHTTP(response, request)
	assert.Equal(http.StatusBadRequest, response.Code)
}

func TestUseID(t *testing.T) {
	t.Run("F", func(t *testing.T) {
		t.Run("NilStrategy", testUseIDFNilStrategy)
		t.Run("Error", testUseIDFError)
	})

	t.Run("FromHeader", func(t *testing.T) {
		testUseIDFromHeader(t)
		t.Run("Missing", testUseIDFromHeaderMissing)
	})

	t.Run("FromPath", func(t *testing.T) {
		testUseIDFromPath(t)
		t.Run("MissingVars", testUseIDFromPathMissingVars)
		t.Run("MissingDeviceNameVar", testUseIDFromPathMissingDeviceNameVar)
	})
}

func testMessageHandlerLogger(t *testing.T) {
	var (
		assert = assert.New(t)
		logger = logging.NewTestLogger(nil, t)

		handler = MessageHandler{}
	)

	assert.NotNil(handler.logger())

	handler.Logger = logger
	assert.Equal(logger, handler.logger())
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
			Router: router,
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

		requestContents []byte
	)

	require.NoError(wrp.NewEncoderBytes(&requestContents, wrp.Msgpack).Encode(message))

	var (
		response           = httptest.NewRecorder()
		request            = httptest.NewRequest("POST", "/foo", bytes.NewReader(requestContents))
		actualResponseBody map[string]interface{}

		router  = new(mockRouter)
		handler = MessageHandler{
			Router: router,
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

		requestContents []byte
	)

	require.NoError(wrp.NewEncoderBytes(&requestContents, requestFormat).Encode(event))

	var (
		response = httptest.NewRecorder()
		request  = httptest.NewRequest("POST", "/foo", bytes.NewReader(requestContents))

		router  = new(mockRouter)
		handler = MessageHandler{
			Logger: logging.NewTestLogger(nil, t),
			Router: router,
		}

		actualDeviceRequest *Request
	)

	request.Header.Set("Content-Type", requestFormat.ContentType())

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

		requestContents  []byte
		responseContents []byte
	)

	require.NoError(wrp.NewEncoderBytes(&requestContents, requestFormat).Encode(requestMessage))
	require.NoError(wrp.NewEncoderBytes(&responseContents, responseFormat).Encode(responseMessage))

	var (
		response = httptest.NewRecorder()
		request  = httptest.NewRequest("POST", "/foo", bytes.NewReader(requestContents))

		router  = new(mockRouter)
		device  = new(MockDevice)
		handler = MessageHandler{
			Logger: logging.NewTestLogger(nil, t),
			Router: router,
		}

		actualDeviceRequest    *Request
		expectedDeviceResponse = &Response{
			Device:   device,
			Message:  responseMessage,
			Format:   wrp.Msgpack,
			Contents: responseContents,
		}
	)

	request.Header.Set("Content-Type", requestFormat.ContentType())
	request.Header.Set("Accept", responseFormat.ContentType())

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
	assert.NoError(wrp.NewDecoder(response.Body, responseFormat).Decode(new(wrp.Message)))

	router.AssertExpectations(t)
	device.AssertExpectations(t)
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

		requestContents []byte
	)

	require.NoError(wrp.NewEncoderBytes(&requestContents, wrp.Msgpack).Encode(requestMessage))

	var (
		response = httptest.NewRecorder()
		request  = httptest.NewRequest("POST", "/foo", bytes.NewReader(requestContents))

		router  = new(mockRouter)
		device  = new(MockDevice)
		handler = MessageHandler{
			Router: router,
		}

		actualResponseBody     map[string]interface{}
		expectedDeviceResponse = &Response{
			Device:  device,
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
	device.AssertExpectations(t)
}

func TestMessageHandler(t *testing.T) {
	t.Run("Logger", testMessageHandlerLogger)

	t.Run("ServeHTTP", func(t *testing.T) {
		t.Run("DecodeError", testMessageHandlerServeHTTPDecodeError)
		t.Run("EncodeError", testMessageHandlerServeHTTPEncodeError)

		t.Run("RouteError", func(t *testing.T) {
			testMessageHandlerServeHTTPRouteError(t, ErrorInvalidDeviceName, http.StatusBadRequest)
			testMessageHandlerServeHTTPRouteError(t, ErrorDeviceNotFound, http.StatusNotFound)
			testMessageHandlerServeHTTPRouteError(t, ErrorNonUniqueID, http.StatusBadRequest)
			testMessageHandlerServeHTTPRouteError(t, ErrorInvalidTransactionKey, http.StatusBadRequest)
			testMessageHandlerServeHTTPRouteError(t, ErrorTransactionAlreadyRegistered, http.StatusBadRequest)
			testMessageHandlerServeHTTPRouteError(t, errors.New("random error"), http.StatusGatewayTimeout)
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
		logger = logging.NewTestLogger(nil, t)

		handler = ConnectHandler{}
	)

	assert.NotNil(handler.logger())

	handler.Logger = logger
	assert.Equal(logger, handler.logger())
}

func testConnectHandlerServeHTTP(t *testing.T, connectError error, responseHeader http.Header) {
	var (
		assert = assert.New(t)

		device    = new(MockDevice)
		connector = new(MockConnector)
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

func testListHandlerRefresh(t *testing.T) {
	var (
		assert  = assert.New(t)
		handler = ListHandler{
			Logger: logging.NewTestLogger(nil, t),
		}
	)

	assert.Equal(DefaultListRefresh, handler.refresh())

	handler.Refresh = 67 * time.Minute
	assert.Equal(67*time.Minute, handler.refresh())
}

func testListHandlerServeHTTP(t *testing.T) {
	var (
		assert              = assert.New(t)
		require             = require.New(t)
		expectedConnectedAt = time.Now().UTC()
		expectedUpTime      = 47913 * time.Minute
		registry            = new(MockRegistry)
		logger              = logging.NewTestLogger(nil, t)

		now = func() time.Time {
			return expectedConnectedAt.Add(expectedUpTime)
		}

		firstDevice  = newDevice(deviceOptions{ID: ID("firat"), QueueSize: 1, ConnectedAt: expectedConnectedAt, Logger: logger})
		secondDevice = newDevice(deviceOptions{ID: ID("second"), QueueSize: 1, ConnectedAt: expectedConnectedAt, Logger: logger})

		handler = ListHandler{
			Logger:   logging.NewTestLogger(nil, t),
			Registry: registry,
		}
	)

	firstDevice.statistics = NewStatistics(now, expectedConnectedAt)
	secondDevice.statistics = NewStatistics(now, expectedConnectedAt)

	registry.On("VisitAll", mock.MatchedBy(func(func(Interface) bool) bool { return true })).Return(0).Once()
	registry.On("VisitAll", mock.MatchedBy(func(func(Interface) bool) bool { return true })).
		Run(func(arguments mock.Arguments) {
			visitor := arguments.Get(0).(func(Interface) bool)
			visitor(firstDevice)
			visitor(secondDevice)
		}).
		Return(0).Once()

	assert.True(handler.cacheExpiry.IsZero())

	{
		var (
			request  = httptest.NewRequest("GET", "/", nil)
			response = httptest.NewRecorder()
		)

		handler.ServeHTTP(response, request)
		assert.Equal(http.StatusOK, response.Code)

		data, err := ioutil.ReadAll(response.Body)
		require.NoError(err)
		assert.JSONEq(`{"devices":[]}`, string(data))

		assert.False(handler.cacheExpiry.IsZero())
		cacheDuration := handler.cacheExpiry.Sub(time.Now())
		assert.True(cacheDuration > 0)
		assert.True(cacheDuration <= handler.refresh(), "The cache duration %s should be less than the refresh interval %s", cacheDuration, handler.refresh())
	}

	expectedJSON := bytes.NewBufferString(`{"devices":[`)
	data, err := firstDevice.MarshalJSON()
	require.NotEmpty(data)
	require.NoError(err)
	expectedJSON.Write(data)

	data, err = secondDevice.MarshalJSON()
	require.NotEmpty(data)
	require.NoError(err)
	expectedJSON.WriteRune(',')
	expectedJSON.Write(data)
	expectedJSON.WriteString(`]}`)

	{
		var (
			request  = httptest.NewRequest("GET", "/", nil)
			response = httptest.NewRecorder()
		)

		handler.cacheExpiry = time.Time{}
		handler.ServeHTTP(response, request)
		assert.Equal(http.StatusOK, response.Code)

		data, err = ioutil.ReadAll(response.Body)
		require.NoError(err)
		assert.JSONEq(expectedJSON.String(), string(data))

		assert.False(handler.cacheExpiry.IsZero())
		cacheDuration := handler.cacheExpiry.Sub(time.Now())
		assert.True(cacheDuration > 0)
		assert.True(cacheDuration <= handler.refresh(), "The cache duration %s should be less than the refresh interval %s", cacheDuration, handler.refresh())
	}

	lastCacheExpiry := handler.cacheExpiry

	{
		var (
			request  = httptest.NewRequest("GET", "/", nil)
			response = httptest.NewRecorder()
		)

		// this should yield the cached bytes
		handler.ServeHTTP(response, request)
		assert.Equal(http.StatusOK, response.Code)

		data, err := ioutil.ReadAll(response.Body)
		require.NoError(err)
		assert.JSONEq(expectedJSON.String(), string(data))
		assert.Equal(lastCacheExpiry, handler.cacheExpiry)
	}

	registry.AssertExpectations(t)
}

func TestListHandler(t *testing.T) {
	t.Run("Refresh", testListHandlerRefresh)
	t.Run("ServeHTTP", testListHandlerServeHTTP)
}

func testStatHandlerNoPathVariables(t *testing.T) {
	var (
		assert   = assert.New(t)
		registry = new(MockRegistry)

		handler = StatHandler{
			Logger:   logging.NewTestLogger(nil, t),
			Registry: registry,
		}

		request  = httptest.NewRequest("GET", "/", nil)
		response = httptest.NewRecorder()
	)

	handler.ServeHTTP(response, request)
	assert.Equal(http.StatusInternalServerError, response.Code)
	registry.AssertExpectations(t)
}

func testStatHandlerNoDeviceName(t *testing.T) {
	var (
		assert   = assert.New(t)
		registry = new(MockRegistry)

		handler = StatHandler{
			Logger:   logging.NewTestLogger(nil, t),
			Registry: registry,
			Variable: "deviceID",
		}

		router   = mux.NewRouter()
		request  = httptest.NewRequest("GET", "/foobar", nil)
		response = httptest.NewRecorder()
	)

	router.Handle("/{doesNotMatter}", &handler)
	router.ServeHTTP(response, request)
	assert.Equal(http.StatusInternalServerError, response.Code)
	registry.AssertExpectations(t)
}

func testStatHandlerInvalidDeviceName(t *testing.T) {
	var (
		assert   = assert.New(t)
		registry = new(MockRegistry)

		handler = StatHandler{
			Logger:   logging.NewTestLogger(nil, t),
			Registry: registry,
			Variable: "deviceID",
		}

		router   = mux.NewRouter()
		request  = httptest.NewRequest("GET", "/asdfqwer:thisisnotvalidasdfasdf", nil)
		response = httptest.NewRecorder()
	)

	router.Handle("/{deviceID}", &handler)
	router.ServeHTTP(response, request)
	assert.Equal(http.StatusBadRequest, response.Code)
	registry.AssertExpectations(t)
}

func testStatHandlerMissingDevice(t *testing.T) {
	var (
		assert   = assert.New(t)
		registry = new(MockRegistry)

		handler = StatHandler{
			Logger:   logging.NewTestLogger(nil, t),
			Registry: registry,
			Variable: "deviceID",
		}

		router   = mux.NewRouter()
		request  = httptest.NewRequest("GET", "/mac:112233445566", nil)
		response = httptest.NewRecorder()
	)

	router.Handle("/{deviceID}", &handler)
	registry.On("Get", ID("mac:112233445566")).Return(nil, false).Once()

	router.ServeHTTP(response, request)
	assert.Equal(http.StatusNotFound, response.Code)
	registry.AssertExpectations(t)
}

func testStatHandlerMarshalJSONFailed(t *testing.T) {
	var (
		assert   = assert.New(t)
		registry = new(MockRegistry)
		device   = new(MockDevice)

		handler = StatHandler{
			Logger:   logging.NewTestLogger(nil, t),
			Registry: registry,
			Variable: "deviceID",
		}

		router   = mux.NewRouter()
		request  = httptest.NewRequest("GET", "/mac:112233445566", nil)
		response = httptest.NewRecorder()
	)

	router.Handle("/{deviceID}", &handler)
	registry.On("Get", ID("mac:112233445566")).Return(device, true).Once()
	device.On("MarshalJSON").Return([]byte{}, errors.New("expected")).Once()

	router.ServeHTTP(response, request)
	assert.Equal(http.StatusInternalServerError, response.Code)
	registry.AssertExpectations(t)
	device.AssertExpectations(t)
}

func testStatHandlerSuccess(t *testing.T) {
	var (
		assert   = assert.New(t)
		registry = new(MockRegistry)
		device   = new(MockDevice)

		handler = StatHandler{
			Logger:   logging.NewTestLogger(nil, t),
			Registry: registry,
			Variable: "deviceID",
		}

		router   = mux.NewRouter()
		request  = httptest.NewRequest("GET", "/mac:112233445566", nil)
		response = httptest.NewRecorder()
	)

	router.Handle("/{deviceID}", &handler)
	registry.On("Get", ID("mac:112233445566")).Return(device, true).Once()
	device.On("MarshalJSON").Return([]byte(`{"foo": "bar"}`), (error)(nil)).Once()

	router.ServeHTTP(response, request)
	assert.Equal(http.StatusOK, response.Code)
	assert.Equal("application/json", response.Header().Get("Content-Type"))
	assert.Equal(`{"foo": "bar"}`, response.Body.String())
	registry.AssertExpectations(t)
	device.AssertExpectations(t)
}

func TestStatHandler(t *testing.T) {
	t.Run("NoPathVariables", testStatHandlerNoPathVariables)
	t.Run("NoDeviceName", testStatHandlerNoDeviceName)
	t.Run("InvalidDeviceName", testStatHandlerInvalidDeviceName)
	t.Run("MissingDevice", testStatHandlerMissingDevice)
	t.Run("MarshalJSONFailed", testStatHandlerMarshalJSONFailed)
	t.Run("Success", testStatHandlerSuccess)
}

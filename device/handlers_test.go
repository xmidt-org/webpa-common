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
			testData := []struct {
				routeError   error
				expectedCode int
			}{
				{ErrorInvalidDeviceName, http.StatusBadRequest},
				{ErrorDeviceNotFound, http.StatusNotFound},
				{ErrorNonUniqueID, http.StatusBadRequest},
				{ErrorInvalidTransactionKey, http.StatusBadRequest},
				{ErrorTransactionAlreadyRegistered, http.StatusBadRequest},
				{errors.New("random error"), http.StatusInternalServerError},
			}

			for _, record := range testData {
				testMessageHandlerServeHTTPRouteError(t, record.routeError, record.expectedCode)
			}
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

package device

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/Comcast/webpa-common/wrp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testRequestContext(t *testing.T) {
	var (
		assert   = assert.New(t)
		message  = new(wrp.Message)
		format   = wrp.JSON
		contents = []byte("some contents")

		request = &Request{
			Message:  message,
			Format:   format,
			Contents: contents,
		}
	)

	assert.Equal(context.Background(), request.Context())
	assert.Panics(func() {
		request.WithContext(nil)
	})

	newContext := context.WithValue(context.Background(), "foo", "bar")
	assert.True(request == request.WithContext(newContext))
	assert.Equal(newContext, request.Context())
}

func testRequestID(t *testing.T) {
	var (
		assert  = assert.New(t)
		request = &Request{
			Message: &wrp.Message{
				Destination: "mac:123412341234",
			},
		}
	)

	id, err := request.ID()
	assert.Equal(ID("mac:123412341234"), id)
	assert.NoError(err)

	request.Message = &wrp.Message{
		Destination: "this is not a valid device ID",
	}

	id, err = request.ID()
	assert.Empty(string(id))
	assert.Error(err)
}

func TestRequest(t *testing.T) {
	t.Run("Context", testRequestContext)
	t.Run("ID", testRequestID)
}

func testDecodeRequest(t *testing.T, message wrp.Routable, format wrp.Format) {
	var (
		assert   = assert.New(t)
		require  = require.New(t)
		contents []byte
		encoders = wrp.NewEncoderPool(1, format)
		decoders = wrp.NewDecoderPool(1, format)
	)

	require.NoError(encoders.EncodeBytes(&contents, message))

	request, err := DecodeRequest(bytes.NewReader(contents), decoders)
	require.NotNil(request)
	require.NoError(err)

	assert.Equal(message.MessageType(), request.Message.MessageType())
	assert.Equal(message.To(), request.Message.To())
	assert.Equal(message.From(), request.Message.From())
	assert.Equal(message.TransactionKey(), request.Message.TransactionKey())
	assert.Equal(format, request.Format)
	assert.Equal(contents, request.Contents)
	assert.Nil(request.ctx)
}

func testDecodeRequestReadError(t *testing.T, format wrp.Format) {
	var (
		assert        = assert.New(t)
		decoders      = wrp.NewDecoderPool(1, format)
		source        = new(mockReader)
		expectedError = errors.New("expected error")
	)

	source.On("Read", mock.AnythingOfType("[]uint8")).Return(0, expectedError)
	request, err := DecodeRequest(source, decoders)
	assert.Nil(request)
	assert.Equal(expectedError, err)

	source.AssertExpectations(t)
}

func testDecodeRequestDecodeError(t *testing.T, format wrp.Format) {
	var (
		assert   = assert.New(t)
		decoders = wrp.NewDecoderPool(1, format)
		empty    []byte
	)

	request, err := DecodeRequest(bytes.NewReader(empty), decoders)
	assert.Nil(request)
	assert.Error(err)
}

func TestDecodeRequest(t *testing.T) {
	for _, format := range []wrp.Format{wrp.Msgpack, wrp.JSON} {
		t.Run(format.String(), func(t *testing.T) {
			testDecodeRequest(
				t,
				&wrp.SimpleEvent{
					Source:      "app.comcast.com:9999",
					Destination: "uuid:1234/service",
					ContentType: "text/plain",
					Payload:     []byte("hi there"),
				},
				format,
			)

			testDecodeRequest(
				t,
				&wrp.SimpleRequestResponse{
					Source:          "app.comcast.com:9999",
					Destination:     "uuid:1234/service",
					TransactionUUID: "this-is-a-transaction-id",
					ContentType:     "text/plain",
					Payload:         []byte("hi there"),
					Metadata:        map[string]string{"foo": "bar"},
				},
				format,
			)
		})
	}

	t.Run("ReadError", func(t *testing.T) {
		for _, format := range []wrp.Format{wrp.Msgpack, wrp.JSON} {
			testDecodeRequestReadError(t, format)
		}
	})

	t.Run("DecodeError", func(t *testing.T) {
		for _, format := range []wrp.Format{wrp.Msgpack, wrp.JSON} {
			testDecodeRequestDecodeError(t, format)
		}
	})
}

func testEncodeResponsePool(t *testing.T, message wrp.Message, responseFormat, poolFormat wrp.Format) {
	const encodedConvey = "expected encoded convey"

	var (
		assert  = assert.New(t)
		require = require.New(t)

		device         = new(mockDevice)
		setupEncoders  = wrp.NewEncoderPool(1, responseFormat)
		pool           = wrp.NewEncoderPool(1, poolFormat)
		verifyDecoders = wrp.NewDecoderPool(1, poolFormat)
		contents       []byte
		httpResponse   = httptest.NewRecorder()
	)

	require.NoError(setupEncoders.EncodeBytes(&contents, &message))
	deviceResponse := &Response{
		Device:   device,
		Message:  &message,
		Format:   responseFormat,
		Contents: contents,
	}

	device.On("EncodedConvey").Once().Return(encodedConvey)

	assert.NoError(EncodeResponse(httpResponse, deviceResponse, pool))
	assert.Equal(http.StatusOK, httpResponse.Code)
	assert.Equal(poolFormat.ContentType(), httpResponse.HeaderMap.Get("Content-Type"))
	assert.Equal(encodedConvey, httpResponse.HeaderMap.Get(ConveyHeader))

	actualMessage := new(wrp.Message)
	assert.NoError(verifyDecoders.Decode(actualMessage, httpResponse.Body))
	assert.Equal(message, *actualMessage)

	device.AssertExpectations(t)
}

func testEncodeResponsePoolAndNoContents(t *testing.T, format wrp.Format) {
	var (
		assert         = assert.New(t)
		actualContents = make(map[string]interface{})
		pool           = wrp.NewEncoderPool(1, format)
		device         = new(mockDevice)

		deviceResponse = &Response{
			Device:  device,
			Message: new(wrp.Message),
			Format:  format,
		}

		httpResponse = httptest.NewRecorder()
	)

	device.On("EncodedConvey").Once().Return("")

	assert.NoError(EncodeResponse(httpResponse, deviceResponse, pool))
	assert.Equal(http.StatusInternalServerError, httpResponse.Code)
	assert.Equal("application/json", httpResponse.HeaderMap.Get("Content-Type"))
	assert.Empty(httpResponse.HeaderMap.Get(ConveyHeader))
	assert.NoError(
		json.Unmarshal(httpResponse.Body.Bytes(), &actualContents),
	)

	device.AssertExpectations(t)
}

func testEncodeResponseNoPool(t *testing.T, message wrp.Message, format wrp.Format) {
	var (
		assert       = assert.New(t)
		require      = require.New(t)
		encoders     = wrp.NewEncoderPool(1, format)
		contents     []byte
		device       = new(mockDevice)
		httpResponse = httptest.NewRecorder()
	)

	device.On("EncodedConvey").Once().Return("")

	require.NoError(encoders.EncodeBytes(&contents, &message))
	deviceResponse := &Response{
		Device:   device,
		Message:  &message,
		Format:   format,
		Contents: contents,
	}

	assert.NoError(EncodeResponse(httpResponse, deviceResponse, nil))
	assert.Equal(http.StatusOK, httpResponse.Code)
	assert.Equal(format.ContentType(), httpResponse.HeaderMap.Get("Content-Type"))
	assert.Empty(httpResponse.HeaderMap.Get(ConveyHeader))
	assert.Equal(contents, httpResponse.Body.Bytes())

	device.AssertExpectations(t)
}

func testEncodeResponseNoPoolAndNoContents(t *testing.T) {
	var (
		assert         = assert.New(t)
		actualContents = make(map[string]interface{})
		device         = new(mockDevice)

		deviceResponse = &Response{
			Device:  device,
			Message: new(wrp.Message),
		}

		httpResponse = httptest.NewRecorder()
	)

	device.On("EncodedConvey").Once().Return("")

	assert.NoError(EncodeResponse(httpResponse, deviceResponse, nil))
	assert.Equal(http.StatusInternalServerError, httpResponse.Code)
	assert.Equal("application/json", httpResponse.HeaderMap.Get("Content-Type"))
	assert.Empty(httpResponse.HeaderMap.Get(ConveyHeader))
	assert.NoError(
		json.Unmarshal(httpResponse.Body.Bytes(), &actualContents),
	)

	device.AssertExpectations(t)
}

func TestEncodeResponse(t *testing.T) {
	testData := []wrp.Message{
		wrp.Message{},
		wrp.Message{
			Type:        wrp.SimpleEventMessageType,
			ContentType: "text/plain",
			Payload:     []byte("here is a payload"),
		},
	}

	t.Run("Pool", func(t *testing.T) {
		for _, responseFormat := range []wrp.Format{wrp.Msgpack, wrp.JSON} {
			for _, poolFormat := range []wrp.Format{wrp.Msgpack, wrp.JSON} {
				for _, message := range testData {
					testEncodeResponsePool(t, message, responseFormat, poolFormat)
				}
			}

			t.Run("NoContents", func(t *testing.T) {
				testEncodeResponsePoolAndNoContents(t, responseFormat)
			})
		}
	})

	t.Run("NoPool", func(t *testing.T) {
		for _, format := range []wrp.Format{wrp.Msgpack, wrp.JSON} {
			for _, message := range testData {
				testEncodeResponseNoPool(t, message, format)
			}
		}

		t.Run("NoContents", testEncodeResponseNoPoolAndNoContents)
	})
}

func testTransactionsInitialState(t *testing.T) {
	var (
		assert       = assert.New(t)
		transactions = NewTransactions()
	)

	assert.Equal(0, transactions.Len())
	assert.Empty(transactions.Keys())
}

func testTransactionsCompleteEmptyTransactionKey(t *testing.T) {
	var (
		assert       = assert.New(t)
		transactions = NewTransactions()
	)

	assert.Equal(ErrorInvalidTransactionKey, transactions.Complete("", &Response{}))
}

func testTransactionsCompleteNoSuchTransactionKey(t *testing.T) {
	var (
		assert       = assert.New(t)
		transactions = NewTransactions()
	)

	assert.Equal(ErrorNoSuchTransactionKey, transactions.Complete("nosuch", &Response{}))
}

func testTransactionsCompleteNilResponse(t *testing.T) {
	var (
		assert       = assert.New(t)
		transactions = NewTransactions()
	)

	assert.Panics(func() {
		transactions.Complete("transaction-uuid", nil)
	})
}

func testTransactionsRegisterEmptyTransactionKey(t *testing.T) {
	var (
		assert       = assert.New(t)
		transactions = NewTransactions()
		output, err  = transactions.Register("")
	)

	assert.Equal(0, transactions.Len())
	assert.Empty(transactions.Keys())
	assert.Nil(output)
	assert.Equal(ErrorInvalidTransactionKey, err)
}

func testTransactionsRegisterDuplicateTransactionKey(t *testing.T) {
	const transactionKey = "valid-transaction-id"

	var (
		assert           = assert.New(t)
		transactions     = NewTransactions()
		firstOutput, err = transactions.Register(transactionKey)
	)

	assert.NotNil(firstOutput)
	assert.NoError(err)

	secondOutput, err := transactions.Register(transactionKey)
	assert.Nil(secondOutput)
	assert.Equal(ErrorTransactionAlreadyRegistered, err)
}

func testTransactionsLifecycle(t *testing.T) {
	const transactionKey = "transaction-id"

	var (
		assert           = assert.New(t)
		transactions     = NewTransactions()
		expectedResponse = new(Response)
		registered       = make(chan struct{})
		finished         = make(chan struct{})
	)

	go func() {
		defer close(finished)
		output, err := transactions.Register(transactionKey)
		assert.Equal(1, transactions.Len())
		assert.Equal([]string{transactionKey}, transactions.Keys())
		close(registered)

		if assert.NotNil(output) && assert.NoError(err) {
			assert.True(expectedResponse == <-output)
		}
	}()

	go func() {
		<-registered
		transactions.Complete(transactionKey, expectedResponse)
	}()

	<-finished
}

func testTransactionsCancellation(t *testing.T) {
	const transactionKey = "transaction-id"

	var (
		assert       = assert.New(t)
		transactions = NewTransactions()
		registered   = make(chan struct{})
		finished     = make(chan struct{})
	)

	go func() {
		defer close(finished)
		output, err := transactions.Register(transactionKey)
		assert.Equal(1, transactions.Len())
		assert.Equal([]string{transactionKey}, transactions.Keys())
		close(registered)

		if assert.NotNil(output) && assert.NoError(err) {
			assert.Nil(<-output)
		}
	}()

	go func() {
		<-registered
		transactions.Cancel(transactionKey)
	}()

	<-finished
}

func TestTransactions(t *testing.T) {
	t.Run("InitialState", testTransactionsInitialState)

	t.Run("Complete", func(t *testing.T) {
		t.Run("EmptyTransactionKey", testTransactionsCompleteEmptyTransactionKey)
		t.Run("NoSuchTransactionKey", testTransactionsCompleteNoSuchTransactionKey)
		t.Run("NilResponse", testTransactionsCompleteNilResponse)
	})

	t.Run("Register", func(t *testing.T) {
		t.Run("EmptyTransactionKey", testTransactionsRegisterEmptyTransactionKey)
		t.Run("DuplicateTransactionKey", testTransactionsRegisterDuplicateTransactionKey)
	})

	t.Run("Lifecycle", testTransactionsLifecycle)
	t.Run("Cancellation", testTransactionsCancellation)
}

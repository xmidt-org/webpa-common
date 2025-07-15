// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package device

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/wrp-go/v3"
)

func testRequestContext(t *testing.T) {
	var (
		assert = assert.New(t)
		// nolint: typecheck
		message = new(wrp.Message)
		// nolint: typecheck
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
		// nolint:staticcheck
		request.WithContext(nil)
	})

	// nolint:staticcheck
	newContext := context.WithValue(context.Background(), "foo", "bar")
	assert.True(request == request.WithContext(newContext))
	assert.Equal(newContext, request.Context())
}

func testRequestID(t *testing.T) {
	var (
		assert  = assert.New(t)
		request = &Request{
			// nolint: typecheck
			Message: &wrp.Message{
				Destination: "mac:123412341234",
			},
		}
	)

	id, err := request.ID()
	assert.Equal(ID("mac:123412341234"), id)
	assert.NoError(err)

	// nolint: typecheck
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

// nolint: typecheck
func testDecodeRequest(t *testing.T, message wrp.Routable, format wrp.Format) {
	var (
		assert   = assert.New(t)
		require  = require.New(t)
		contents []byte
	)

	// nolint: typecheck
	require.NoError(wrp.NewEncoderBytes(&contents, format).Encode(message))

	request, err := DecodeRequest(bytes.NewReader(contents), format)
	require.NotNil(request)
	require.NoError(err)

	// nolint: typecheck
	if routable, ok := request.Message.(wrp.Routable); ok {
		assert.Equal(message.MessageType(), routable.MessageType())
		assert.Equal(message.To(), routable.To())
		assert.Equal(message.From(), routable.From())
		assert.Equal(message.TransactionKey(), routable.TransactionKey())
	}

	assert.Equal(format, request.Format)
	assert.Equal(contents, request.Contents)
	assert.Nil(request.ctx)
}

// nolint: typecheck
func testDecodeRequestReadError(t *testing.T, format wrp.Format) {
	var (
		assert        = assert.New(t)
		source        = new(mockReader)
		expectedError = errors.New("expected error")
	)

	// nolint: typecheck
	source.On("Read", mock.AnythingOfType("[]uint8")).Return(0, expectedError)
	request, err := DecodeRequest(source, format)
	assert.Nil(request)
	assert.Equal(expectedError, err)

	// nolint: typecheck
	source.AssertExpectations(t)
}

// nolint: typecheck
func testDecodeRequestDecodeError(t *testing.T, format wrp.Format) {
	var (
		assert = assert.New(t)
		empty  []byte
	)

	request, err := DecodeRequest(bytes.NewReader(empty), format)
	assert.Nil(request)
	assert.Error(err)
}

func TestDecodeRequest(t *testing.T) {
	// nolint: typecheck
	for _, format := range []wrp.Format{wrp.Msgpack, wrp.JSON} {
		t.Run(format.String(), func(t *testing.T) {
			testDecodeRequest(
				t,
				// nolint: typecheck
				&wrp.Message{
					Source:      "app.comcast.com:9999",
					Destination: "uuid:1234/service",
					ContentType: "text/plain",
					Payload:     []byte("hi there"),
				},
				format,
			)

			testDecodeRequest(
				t,
				// nolint: typecheck
				&wrp.Message{
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
		// nolint: typecheck
		for _, format := range []wrp.Format{wrp.Msgpack, wrp.JSON} {
			testDecodeRequestReadError(t, format)
		}
	})

	t.Run("DecodeError", func(t *testing.T) {
		// nolint: typecheck
		for _, format := range []wrp.Format{wrp.Msgpack, wrp.JSON} {
			testDecodeRequestDecodeError(t, format)
		}
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

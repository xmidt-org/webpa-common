package wrphttp

import (
	"bytes"
	"context"
	"errors"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Comcast/webpa-common/wrp"
	"github.com/Comcast/webpa-common/xhttp/xhttptest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultDecoder(t *testing.T) {
	assert := assert.New(t)
	assert.NotNil(DefaultDecoder())
}

func testDecodeEntitySuccess(t *testing.T) {
	testData := []struct {
		defaultFormat wrp.Format
		bodyFormat    wrp.Format
		contentType   string
	}{
		{wrp.Msgpack, wrp.Msgpack, ""},
		{wrp.JSON, wrp.JSON, ""},
		{wrp.Msgpack, wrp.JSON, wrp.JSON.ContentType()},
		{wrp.JSON, wrp.Msgpack, wrp.Msgpack.ContentType()},
	}

	for i, record := range testData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)

				expected = wrp.Message{
					Source:      "foo",
					Destination: "bar",
				}

				body    bytes.Buffer
				request = httptest.NewRequest("POST", "/", &body)
				decoder = DecodeEntity(record.defaultFormat)
			)

			require.NotNil(decoder)
			require.NoError(
				wrp.NewEncoder(&body, record.bodyFormat).Encode(&expected),
			)

			request.Header.Set("Content-Type", record.contentType)
			entity, err := decoder(context.Background(), request)
			assert.NoError(err)
			require.NotNil(entity)

			assert.Equal(expected, entity.Message)
		})
	}
}

func testDecodeEntityInvalidContentType(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		decoder = DecodeEntity(wrp.Msgpack)
		request = httptest.NewRequest("GET", "/", nil)
	)

	require.NotNil(decoder)
	request.Header.Set("Content-Type", "invalid")
	entity, err := decoder(context.Background(), request)
	assert.Nil(entity)
	assert.Error(err)
}

func testDecodeEntityBodyError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedError = errors.New("expected")
		decoder       = DecodeEntity(wrp.Msgpack)
		body          = new(xhttptest.MockBody)
		request       = httptest.NewRequest("GET", "/", body)
	)

	require.NotNil(decoder)
	body.OnReadError(expectedError).Once()
	entity, err := decoder(context.Background(), request)
	assert.Nil(entity)
	assert.Equal(expectedError, err)

	body.AssertExpectations(t)
}

func TestDecodeEntity(t *testing.T) {
	t.Run("Success", testDecodeEntitySuccess)
	t.Run("InvalidContentType", testDecodeEntityInvalidContentType)
	t.Run("BodyError", testDecodeEntityBodyError)
}

func testDecodeRequestHeadersSuccess(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expected = wrp.Message{
			Type:            wrp.SimpleEventMessageType,
			Source:          "foo",
			Destination:     "bar",
			ContentType:     "application/octet-stream",
			Payload:         []byte{1, 2, 3},
			TransactionUUID: "testytest",
		}

		body    bytes.Buffer
		request = httptest.NewRequest("POST", "/", &body)
	)

	body.Write([]byte{1, 2, 3})
	request.Header.Set(MessageTypeHeader, "event")
	request.Header.Set(SourceHeader, "foo")
	request.Header.Set(DestinationHeader, "bar")
	request.Header.Set(TransactionUuidHeader, "testytest")
	entity, err := DecodeRequestHeaders(context.Background(), request)
	assert.NoError(err)
	require.NotNil(entity)

	assert.Equal(expected, entity.Message)
}

func testDecodeRequestHeadersInvalid(t *testing.T) {
	var (
		assert  = assert.New(t)
		request = httptest.NewRequest("POST", "/", nil)
	)

	request.Header.Set(MessageTypeHeader, "askdjfa;skdjfasdf")
	entity, err := DecodeRequestHeaders(context.Background(), request)
	assert.Nil(entity)
	assert.Error(err)
}

func TestDecodeRequestHeaders(t *testing.T) {
	t.Run("Success", testDecodeRequestHeadersSuccess)
	t.Run("Invalid", testDecodeRequestHeadersInvalid)
}

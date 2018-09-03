package wrphttp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Comcast/webpa-common/wrp"
	gokithttp "github.com/go-kit/kit/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHandlerFunc(t *testing.T) {
	var (
		assert = assert.New(t)

		expectedResponse ResponseWriter = &entityResponseWriter{}
		expectedRequest                 = new(Request)

		called             = false
		hf     HandlerFunc = func(actualResponse ResponseWriter, actualRequest *Request) {
			called = true
			assert.Equal(expectedResponse, actualResponse)
			assert.Equal(expectedRequest, actualRequest)
		}
	)

	hf.ServeWRP(expectedResponse, expectedRequest)
	assert.True(called)
}

func testWithErrorEncoderDefault(t *testing.T) {
	var (
		assert = assert.New(t)
		wh     = new(wrpHandler)
	)

	WithErrorEncoder(nil)(wh)
	assert.NotNil(wh.errorEncoder)
}

func testWithErrorEncoderCustom(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		customCalled                        = false
		custom       gokithttp.ErrorEncoder = func(context.Context, error, http.ResponseWriter) {
			customCalled = true
		}

		wh = new(wrpHandler)
	)

	WithErrorEncoder(custom)(wh)
	require.NotNil(wh.errorEncoder)

	wh.errorEncoder(context.Background(), errors.New("expected"), httptest.NewRecorder())
	assert.True(customCalled)
}

func TestWithErrorEncoder(t *testing.T) {
	t.Run("Default", testWithErrorEncoderDefault)
	t.Run("Custom", testWithErrorEncoderCustom)
}

func testWithNewResponseWriterDefault(t *testing.T) {
	var (
		assert = assert.New(t)
		wh     = new(wrpHandler)
	)

	WithNewResponseWriter(nil)(wh)
	assert.NotNil(wh.newResponseWriter)
}

func testWithNewResponseWriterCustom(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expected                    = &entityResponseWriter{}
		custom   ResponseWriterFunc = func(http.ResponseWriter, *Request) (ResponseWriter, error) {
			return expected, nil
		}

		wh = new(wrpHandler)
	)

	WithNewResponseWriter(custom)(wh)
	require.NotNil(wh.newResponseWriter)

	actual, err := wh.newResponseWriter(httptest.NewRecorder(), new(Request))
	assert.Equal(expected, actual)
	assert.NoError(err)
}

func TestWithNewResponseWriter(t *testing.T) {
	t.Run("Default", testWithNewResponseWriterDefault)
	t.Run("Custom", testWithNewResponseWriterCustom)
}

func testWithDecoderDefault(t *testing.T) {
	var (
		assert = assert.New(t)
		wh     = new(wrpHandler)
	)

	WithDecoder(nil)(wh)
	assert.NotNil(wh.decoder)
}

func testWithDecoderCustom(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expected         = new(Entity)
		custom   Decoder = func(context.Context, *http.Request) (*Entity, error) {
			return expected, nil
		}

		wh = new(wrpHandler)
	)

	WithDecoder(custom)(wh)
	require.NotNil(wh.decoder)

	actual, err := wh.decoder(context.Background(), httptest.NewRequest("GET", "/", nil))
	assert.Equal(expected, actual)
	assert.NoError(err)
}

func TestWithDecoder(t *testing.T) {
	t.Run("Default", testWithDecoderDefault)
	t.Run("Custom", testWithDecoderCustom)
}

func testWithBeforeNone(t *testing.T) {
	var (
		assert = assert.New(t)
		wh     = new(wrpHandler)
	)

	WithBefore(nil)(wh)
	assert.Empty(wh.before)

	WithBefore([]MessageFunc{}...)(wh)
	assert.Empty(wh.before)
}

func TestWithBefore(t *testing.T) {
	testData := [][]MessageFunc{
		nil,
		[]MessageFunc{},
		[]MessageFunc{
			func(context.Context, *wrp.Message) context.Context { return nil },
		},
		[]MessageFunc{
			func(context.Context, *wrp.Message) context.Context { return nil },
			func(context.Context, *wrp.Message) context.Context { return nil },
			func(context.Context, *wrp.Message) context.Context { return nil },
		},
	}

	for i, record := range testData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var (
				assert = assert.New(t)
				wh     = new(wrpHandler)
			)

			WithBefore(record...)(wh)
			assert.Len(wh.before, len(record))
			WithBefore(record...)(wh)
			assert.Len(wh.before, 2*len(record))
		})
	}
}

func TestNewHTTPHandler(t *testing.T) {
	assert := assert.New(t)
	assert.Panics(func() {
		NewHTTPHandler(nil)
	})
}

func testWRPHandlerDecodeError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedCtx = context.WithValue(context.Background(), "foo", "bar")
		expectedErr = errors.New("expected")

		decoder = func(actualCtx context.Context, httpRequest *http.Request) (*Entity, error) {
			assert.Equal(expectedCtx, actualCtx)
			return nil, expectedErr
		}

		errorEncoderCalled = false
		errorEncoder       = func(actualCtx context.Context, actualErr error, _ http.ResponseWriter) {
			errorEncoderCalled = true
			assert.Equal(expectedCtx, actualCtx)
			assert.Equal(expectedErr, actualErr)
		}

		wrpHandler  = new(MockHandler)
		httpHandler = NewHTTPHandler(wrpHandler, WithDecoder(decoder), WithErrorEncoder(errorEncoder))

		httpResponse = httptest.NewRecorder()
		httpRequest  = httptest.NewRequest("POST", "/", nil).WithContext(expectedCtx)
	)

	require.NotNil(httpHandler)
	httpHandler.ServeHTTP(httpResponse, httpRequest)

	assert.True(errorEncoderCalled)
	wrpHandler.AssertExpectations(t)
}

func testWRPHandlerResponseWriterError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedCtx    = context.WithValue(context.Background(), "foo", "bar")
		expectedErr    = errors.New("expected")
		expectedEntity = &Entity{
			Message: wrp.Message{
				Type: wrp.SimpleEventMessageType,
			},
		}

		before = func(ctx context.Context, m *wrp.Message) context.Context {
			m.ContentType = "something"
			return ctx
		}

		decoder = func(actualCtx context.Context, httpRequest *http.Request) (*Entity, error) {
			assert.Equal(expectedCtx, actualCtx)
			return expectedEntity, nil
		}

		newResponseWriterCalled = false
		newResponseWriter       = func(_ http.ResponseWriter, wrpRequest *Request) (ResponseWriter, error) {
			newResponseWriterCalled = true
			assert.Equal(
				wrp.Message{
					Type:        wrp.SimpleEventMessageType,
					ContentType: "something",
				},
				wrpRequest.Entity.Message,
			)

			return nil, expectedErr
		}

		errorEncoderCalled = false
		errorEncoder       = func(actualCtx context.Context, actualErr error, _ http.ResponseWriter) {
			errorEncoderCalled = true
			assert.Equal(expectedCtx, actualCtx)
			assert.Equal(expectedErr, actualErr)
		}

		wrpHandler  = new(MockHandler)
		httpHandler = NewHTTPHandler(wrpHandler,
			WithBefore(before),
			WithDecoder(decoder),
			WithNewResponseWriter(newResponseWriter),
			WithErrorEncoder(errorEncoder),
		)

		httpResponse = httptest.NewRecorder()
		httpRequest  = httptest.NewRequest("POST", "/", nil).WithContext(expectedCtx)
	)

	require.NotNil(httpHandler)
	httpHandler.ServeHTTP(httpResponse, httpRequest)

	assert.True(newResponseWriterCalled)
	assert.True(errorEncoderCalled)
	wrpHandler.AssertExpectations(t)
}

func testWRPHandlerSuccess(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedCtx    = context.WithValue(context.Background(), "foo", "bar")
		expectedEntity = &Entity{
			Message: wrp.Message{
				Type: wrp.SimpleEventMessageType,
			},
		}

		before = func(ctx context.Context, m *wrp.Message) context.Context {
			m.ContentType = "something"
			return ctx
		}

		decoder = func(actualCtx context.Context, httpRequest *http.Request) (*Entity, error) {
			assert.Equal(expectedCtx, actualCtx)
			return expectedEntity, nil
		}

		wrpHandler  = new(MockHandler)
		httpHandler = NewHTTPHandler(wrpHandler,
			WithBefore(before),
			WithDecoder(decoder),
			WithNewResponseWriter(NewEntityResponseWriter(wrp.Msgpack)),
		)

		httpResponse = httptest.NewRecorder()
		httpRequest  = httptest.NewRequest("POST", "/", nil).WithContext(expectedCtx)
	)

	require.NotNil(httpHandler)
	wrpHandler.On("ServeWRP",
		mock.MatchedBy(func(r ResponseWriter) bool {
			return r != nil
		}),
		mock.MatchedBy(func(r *Request) bool {
			return assert.Equal(wrp.Message{Type: wrp.SimpleEventMessageType, ContentType: "something"}, r.Entity.Message)
		}),
	).Once()

	httpHandler.ServeHTTP(httpResponse, httpRequest)
	wrpHandler.AssertExpectations(t)
}

func TestWRPHandler(t *testing.T) {
	t.Run("ServeHTTP", func(t *testing.T) {
		t.Run("DecodeError", testWRPHandlerDecodeError)
		t.Run("ResponseWriterError", testWRPHandlerResponseWriterError)
		t.Run("Success", testWRPHandlerSuccess)
	})
}

package bookkeeping

import (
	"net/http"
	"net/http/httptest"
	"testing"

	gokithttp "github.com/go-kit/kit/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/logging/logginghttp"
	"github.com/xmidt-org/webpa-common/xhttp/xcontext"
)

func TestEmptyBookkeeper(t *testing.T) {
	var (
		assert           = assert.New(t)
		require          = require.New(t)
		transactorCalled = false

		bookkeeper = New()
		logger     = logging.NewCaptureLogger()
	)
	require.NotNil(bookkeeper)
	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(logging.WithLogger(req.Context(), logger))

	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		transactorCalled = true
		writer.Write([]byte("payload"))
		writer.WriteHeader(200)
	})
	rr := httptest.NewRecorder()

	bookkeeper(handler).ServeHTTP(rr, req)
	assert.True(transactorCalled)

	select {
	case result := <-logger.Output():
		assert.Len(result, 4)
	default:
		assert.Fail("CaptureLogger must capture something")

	}
}

func TestBookkeeper(t *testing.T) {
	var (
		assert           = assert.New(t)
		require          = require.New(t)
		transactorCalled = false

		bookkeeper = New(WithResponses(Code))
		logger     = logging.NewCaptureLogger()
	)

	require.NotNil(bookkeeper)
	req := httptest.NewRequest("GET", "/", nil)

	req = req.WithContext(logging.WithLogger(req.Context(), logger))

	rr := httptest.NewRecorder()

	customLogInfo := xcontext.Populate(
		logginghttp.SetLogger(logger,
			logginghttp.RequestInfo,
		),
		gokithttp.PopulateRequestContext,
	)

	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		transactorCalled = true
		writer.Write([]byte("payload"))
		writer.WriteHeader(200)
	})

	bookkeeper(customLogInfo(handler)).ServeHTTP(rr, req)

	assert.True(transactorCalled)

	select {
	case result := <-logger.Output():
		assert.Len(result, 8)
		assert.Equal(req.RequestURI, result["requestURI"])
		assert.Equal(200, result["code"])
	default:
		assert.Fail("CaptureLogger must capture something")

	}
}

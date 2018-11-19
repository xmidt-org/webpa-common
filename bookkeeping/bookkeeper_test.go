package bookkeeping

import (
	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)


func TestEmptyBookkeeper(t *testing.T) {
	var (
		assert           = assert.New(t)
		require          = require.New(t)
		transactorCalled = false

		transactor = func(*http.Request) (*http.Response, error) {
			transactorCalled = true
			return nil, nil
		}
		bookkeeper = Transactor(transactor)
		logger     = logging.NewCaptureLogger()
	)
	require.NotNil(bookkeeper)
	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(logging.WithLogger(req.Context(), logger))

	bookkeeper(req)
	assert.True(transactorCalled)

	select {
	case result := <-logger.Output():
		assert.Len(result, 2)
	default:
		assert.Fail("CaptureLogger must capture something")

	}
}

func TestBookkeeper(t *testing.T) {
	var (
		assert           = assert.New(t)
		require          = require.New(t)
		transactorCalled = false

		transactor = func(*http.Request) (*http.Response, error) {
			transactorCalled = true
			return &http.Response{StatusCode:200}, nil
		}
		bookkeeper = Transactor(transactor, WithRequests(Path), WithResponses(Code))
		logger     = logging.NewCaptureLogger()
	)
	require.NotNil(bookkeeper)
	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(logging.WithLogger(req.Context(), logger))

	bookkeeper(req)
	assert.True(transactorCalled)

	select {
	case result := <-logger.Output():
		assert.Equal(req.URL.Path, result["path"])
		assert.Equal(200, result["code"])
	default:
		assert.Fail("CaptureLogger must capture something")

	}
}

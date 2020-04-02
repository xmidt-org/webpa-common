package drain

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/webpa-common/logging"
)

func testStartServeHTTPDefaultLogger(t *testing.T) {
	var (
		assert = assert.New(t)

		d                     = new(mockDrainer)
		done  <-chan struct{} = make(chan struct{})
		start                 = Start{d}

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("POST", "/", nil)
	)

	d.On("Start", Job{}).Return(done, Job{Count: 126, Percent: 10, Rate: 12, Tick: 5 * time.Minute}, error(nil))
	start.ServeHTTP(response, request)
	assert.Equal(http.StatusOK, response.Code)
	assert.Equal("application/json", response.HeaderMap.Get("Content-Type"))
	assert.JSONEq(
		`{"count": 126, "percent": 10, "rate": 12, "tick": "5m0s"}`,
		response.Body.String(),
	)

	d.AssertExpectations(t)
}

func testStartServeHTTPValid(t *testing.T) {
	testData := []struct {
		uri      string
		expected Job
	}{
		{
			"/foo",
			Job{},
		},
		{
			"/foo?count=100",
			Job{Count: 100},
		},
		{
			"/foo?rate=10",
			Job{Rate: 10},
		},
		{
			"/foo?rate=23&tick=1m",
			Job{Rate: 23, Tick: time.Minute},
		},
		{
			"/foo?count=22&rate=10&tick=20s",
			Job{Count: 22, Rate: 10, Tick: 20 * time.Second},
		},
	}

	for _, record := range testData {
		t.Run(record.uri, func(t *testing.T) {
			var (
				assert = assert.New(t)

				d                     = new(mockDrainer)
				done  <-chan struct{} = make(chan struct{})
				start                 = Start{d}

				ctx      = logging.WithLogger(context.Background(), logging.DefaultLogger())
				response = httptest.NewRecorder()
				request  = httptest.NewRequest("POST", record.uri, nil).WithContext(ctx)
			)

			d.On("Start", record.expected).Return(done, Job{Count: 47192, Percent: 57, Rate: 500, Tick: 37 * time.Second}, error(nil)).Once()
			start.ServeHTTP(response, request)
			assert.Equal(http.StatusOK, response.Code)
			assert.Equal("application/json", response.HeaderMap.Get("Content-Type"))
			assert.JSONEq(
				`{"count": 47192, "percent": 57, "rate": 500, "tick": "37s"}`,
				response.Body.String(),
			)

			d.AssertExpectations(t)
		})
	}
}

func testStartServeHTTPParseFormError(t *testing.T) {
	var (
		assert = assert.New(t)

		d     = new(mockDrainer)
		start = Start{d}

		ctx      = logging.WithLogger(context.Background(), logging.DefaultLogger())
		response = httptest.NewRecorder()
		request  = httptest.NewRequest("POST", "/foo?%TT*&&", nil).WithContext(ctx)
	)

	start.ServeHTTP(response, request)
	assert.Equal(http.StatusBadRequest, response.Code)
	d.AssertExpectations(t)
}

func testStartServeHTTPInvalidQuery(t *testing.T) {
	var (
		assert = assert.New(t)

		d     = new(mockDrainer)
		start = Start{d}

		ctx      = logging.WithLogger(context.Background(), logging.DefaultLogger())
		response = httptest.NewRecorder()
		request  = httptest.NewRequest("POST", "/foo?count=asdf", nil).WithContext(ctx)
	)

	start.ServeHTTP(response, request)
	assert.Equal(http.StatusBadRequest, response.Code)
	d.AssertExpectations(t)
}

func testStartServeHTTPStartError(t *testing.T) {
	var (
		assert = assert.New(t)

		d             = new(mockDrainer)
		done          <-chan struct{}
		start         = Start{d}
		expectedError = errors.New("expected")

		ctx      = logging.WithLogger(context.Background(), logging.DefaultLogger())
		response = httptest.NewRecorder()
		request  = httptest.NewRequest("POST", "/foo?count=100", nil).WithContext(ctx)
	)

	d.On("Start", Job{Count: 100}).Return(done, Job{}, expectedError).Once()
	start.ServeHTTP(response, request)
	assert.Equal(http.StatusConflict, response.Code)
	d.AssertExpectations(t)
}

func TestStart(t *testing.T) {
	t.Run("ServeHTTP", func(t *testing.T) {
		t.Run("DefaultLogger", testStartServeHTTPDefaultLogger)
		t.Run("Valid", testStartServeHTTPValid)
		t.Run("ParseFormError", testStartServeHTTPParseFormError)
		t.Run("InvalidQuery", testStartServeHTTPInvalidQuery)
		t.Run("StartError", testStartServeHTTPStartError)
	})
}

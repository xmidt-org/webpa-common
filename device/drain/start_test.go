package drain

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
)

func testStartServeHTTPDefaultLogger(t *testing.T) {
	var (
		assert = assert.New(t)

		d                     = new(mockDrainer)
		done  <-chan struct{} = make(chan struct{})
		start                 = Start{Drainer: d}

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("POST", "/", nil)
	)

	d.On("Start", Job{}).Return(done, error(nil))
	start.ServeHTTP(response, request)
	assert.Equal(http.StatusOK, response.Code)
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
				start                 = Start{Logger: logging.NewTestLogger(nil, t), Drainer: d}

				response = httptest.NewRecorder()
				request  = httptest.NewRequest("POST", record.uri, nil)
			)

			d.On("Start", record.expected).Return(done, error(nil)).Once()
			start.ServeHTTP(response, request)
			assert.Equal(http.StatusOK, response.Code)
			d.AssertExpectations(t)
		})
	}
}

func testStartServeHTTPParseFormError(t *testing.T) {
	var (
		assert = assert.New(t)

		d     = new(mockDrainer)
		start = Start{Logger: logging.NewTestLogger(nil, t), Drainer: d}

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("POST", "/foo?%TT*&&", nil)
	)

	start.ServeHTTP(response, request)
	assert.Equal(http.StatusBadRequest, response.Code)
	d.AssertExpectations(t)
}

func testStartServeHTTPInvalidQuery(t *testing.T) {
	var (
		assert = assert.New(t)

		d     = new(mockDrainer)
		start = Start{Logger: logging.NewTestLogger(nil, t), Drainer: d}

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("POST", "/foo?count=asdf", nil)
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
		start         = Start{Logger: logging.NewTestLogger(nil, t), Drainer: d}
		expectedError = errors.New("expected")

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("POST", "/foo?count=100", nil)
	)

	d.On("Start", Job{Count: 100}).Return(done, expectedError).Once()
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

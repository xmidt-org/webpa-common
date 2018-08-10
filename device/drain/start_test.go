package drain

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
)

func testStartServeHTTPDefault(t *testing.T) {
	var (
		assert = assert.New(t)

		d     = new(mockDrainer)
		start = Start{Drainer: d}

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("POST", "/", nil)
	)

	d.On("Start", Job{}).Return(error(nil)).Once()
	start.ServeHTTP(response, request)
	assert.Equal(http.StatusOK, response.Code)
	d.AssertExpectations(t)
}

func testStartServeHTTPDrain(t *testing.T) {
	testData := []struct {
		uri      string
		expected Job
	}{
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

				d     = new(mockDrainer)
				start = Start{Logger: logging.NewTestLogger(nil, t), Drainer: d}

				response = httptest.NewRecorder()
				request  = httptest.NewRequest("POST", record.uri, nil)
			)

			d.On("Start", record.expected).Return(error(nil)).Once()
			start.ServeHTTP(response, request)
			assert.Equal(http.StatusOK, response.Code)
			d.AssertExpectations(t)
		})
	}
}

func TestStart(t *testing.T) {
	t.Run("ServeHTTP", func(t *testing.T) {
		t.Run("Default", testStartServeHTTPDefault)
		t.Run("Drain", testStartServeHTTPDrain)
	})
}

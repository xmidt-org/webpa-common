package drain

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/xmidt-org/sallust"
	"github.com/xmidt-org/webpa-common/v2/device/devicegate"

	"github.com/stretchr/testify/assert"
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
	assert.Equal("application/json", response.Header().Get("Content-Type"))
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
			uri:      "/foo",
			expected: Job{},
		},
		{
			uri:      "/foo?count=100",
			expected: Job{Count: 100},
		},
		{
			uri:      "/foo?rate=10",
			expected: Job{Rate: 10},
		},
		{
			uri:      "/foo?rate=23&tick=1m",
			expected: Job{Rate: 23, Tick: time.Minute},
		},
		{
			uri:      "/foo?count=22&rate=10&tick=20s",
			expected: Job{Count: 22, Rate: 10, Tick: 20 * time.Second},
		},
	}

	for _, record := range testData {
		t.Run(record.uri, func(t *testing.T) {
			var (
				assert = assert.New(t)

				d                     = new(mockDrainer)
				done  <-chan struct{} = make(chan struct{})
				start                 = Start{d}

				ctx      = sallust.With(context.Background(), sallust.Default())
				response = httptest.NewRecorder()
				request  = httptest.NewRequest("POST", record.uri, nil).WithContext(ctx)
			)

			d.On("Start", record.expected).Return(done, Job{Count: 47192, Percent: 57, Rate: 500, Tick: 37 * time.Second, DrainFilter: record.expected.DrainFilter}, error(nil)).Once()
			start.ServeHTTP(response, request)
			assert.Equal(http.StatusOK, response.Code)
			assert.Equal("application/json", response.Header().Get("Content-Type"))
			assert.JSONEq(
				`{"count": 47192, "percent": 57, "rate": 500, "tick": "37s"}`,
				response.Body.String(),
			)
			d.AssertExpectations(t)
		})
	}
}

func testStartServeHTTPWithBody(t *testing.T) {
	df := &drainFilter{
		filter: &devicegate.FilterGate{
			FilterStore: devicegate.FilterStore(map[string]devicegate.Set{
				"test": &devicegate.FilterSet{Set: map[interface{}]bool{
					"test1": true,
					"test2": true,
				}},
			}),
		},
		filterRequest: devicegate.FilterRequest{
			Key:    "test",
			Values: []interface{}{"test1", "test2"},
		},
	}

	testData := []struct {
		description        string
		body               []byte
		expected           Job
		expectedJSON       string
		expectedStatusCode int
	}{
		{
			description:        "Success with body",
			body:               []byte(`{"key": "test", "values": ["test1", "test2"]}`),
			expected:           Job{Count: 22, Rate: 10, Tick: 20 * time.Second, DrainFilter: df},
			expectedJSON:       `{"count": 47192, "percent": 57, "rate": 500, "tick": "37s", "filter":{"key": "test", "values": ["test1", "test2"]}}`,
			expectedStatusCode: http.StatusOK,
		},
		{
			description:        "Unmarshal error",
			body:               []byte(`this is not a filter request`),
			expected:           Job{Count: 22, Rate: 10, Tick: 20 * time.Second},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			description:        "Empty Body",
			body:               []byte(`{}`),
			expected:           Job{Count: 22, Rate: 10, Tick: 20 * time.Second},
			expectedStatusCode: http.StatusOK,
		},
		{
			description:        "No value field",
			body:               []byte(`{"key": "test"}`),
			expected:           Job{Count: 22, Rate: 10, Tick: 20 * time.Second},
			expectedStatusCode: http.StatusOK,
		},
		{
			description:        "No key",
			body:               []byte(`{"values": ["test1", "test2"]}`),
			expected:           Job{Count: 22, Rate: 10, Tick: 20 * time.Second},
			expectedStatusCode: http.StatusOK,
		},
		{
			description:        "Empty values array",
			body:               []byte(`{"key": "test", "values": []}`),
			expected:           Job{Count: 22, Rate: 10, Tick: 20 * time.Second},
			expectedStatusCode: http.StatusOK,
		},
	}

	for _, record := range testData {
		t.Run(record.description, func(t *testing.T) {
			var (
				assert = assert.New(t)

				d                     = new(mockDrainer)
				done  <-chan struct{} = make(chan struct{})
				start                 = Start{d}

				ctx      = sallust.With(context.Background(), sallust.Default())
				response = httptest.NewRecorder()
				request  = httptest.NewRequest("POST", "/foo?count=22&rate=10&tick=20s", bytes.NewBuffer(record.body)).WithContext(ctx)
			)

			if record.expectedStatusCode == http.StatusOK {
				d.On("Start", record.expected).Return(done, Job{Count: 47192, Percent: 57, Rate: 500, Tick: 37 * time.Second, DrainFilter: record.expected.DrainFilter}, error(nil)).Once()
			}
			start.ServeHTTP(response, request)
			assert.Equal(record.expectedStatusCode, response.Code)
			assert.Equal("application/json", response.Header().Get("Content-Type"))
			if record.expectedStatusCode == http.StatusOK {
				if len(record.expectedJSON) == 0 {
					assert.JSONEq(
						`{"count": 47192, "percent": 57, "rate": 500, "tick": "37s"}`,
						response.Body.String(),
					)
				} else {
					assert.JSONEq(record.expectedJSON, response.Body.String())
				}
			}

			d.AssertExpectations(t)
		})
	}

}

func testStartServeHTTPParseFormError(t *testing.T) {
	var (
		assert = assert.New(t)

		d     = new(mockDrainer)
		start = Start{d}

		ctx      = sallust.With(context.Background(), sallust.Default())
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

		ctx      = sallust.With(context.Background(), sallust.Default())
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

		ctx      = sallust.With(context.Background(), sallust.Default())
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
		t.Run("WithBody", testStartServeHTTPWithBody)
		t.Run("ParseFormError", testStartServeHTTPParseFormError)
		t.Run("InvalidQuery", testStartServeHTTPInvalidQuery)
		t.Run("StartError", testStartServeHTTPStartError)
	})
}

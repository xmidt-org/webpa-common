package devicegate

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/webpa-common/logging"
)

func TestServeHTTPGet(t *testing.T) {
	var (
		assert = assert.New(t)
		logger = logging.NewTestLogger(nil, t)
		ctx    = logging.WithLogger(context.Background(), logger)

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "/", nil)

		mockDeviceGate = new(mockDeviceGate)

		f = FilterHandler{
			Gate: mockDeviceGate,
		}
	)

	mockDeviceGate.On("VisitAll", mock.Anything).Return(0)

	f.ServeHTTP(response, request.WithContext(ctx))
	assert.Equal(http.StatusOK, response.Code)
	assert.NotEmpty(response.Result().Body)
}

func TestBadRequest(t *testing.T) {
	var (
		logger = logging.NewTestLogger(nil, t)
		ctx    = logging.WithLogger(context.Background(), logger)

		mockDeviceGate = new(mockDeviceGate)
		f              = FilterHandler{
			Gate: mockDeviceGate,
		}
	)

	tests := []struct {
		description        string
		reqBody            []byte
		expectedStatusCode int
		testDelete         bool
	}{
		{
			description:        "Unmarshal error",
			reqBody:            []byte(`this is not a filter request`),
			expectedStatusCode: http.StatusBadRequest,
			testDelete:         true,
		},
		{
			description:        "No filter key parameter",
			reqBody:            []byte(`{"test": "test"}`),
			expectedStatusCode: http.StatusBadRequest,
			testDelete:         true,
		},
		{
			description:        "No filter values",
			reqBody:            []byte(`{"key": "test"}`),
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			description:        "Filter key not allowed",
			reqBody:            []byte(`{"key": "test", "values": ["test", "test1"]}`),
			expectedStatusCode: http.StatusBadRequest,
		},
	}

	mockDeviceGate.On("GetAllowedFilters").Return(make(FilterSet), true)

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)

			requests := []*http.Request{
				httptest.NewRequest("POST", "/", bytes.NewBuffer(tc.reqBody)),
				httptest.NewRequest("PUT", "/", bytes.NewBuffer(tc.reqBody)),
			}

			if tc.testDelete {
				requests = append(requests, httptest.NewRequest("DELETE", "/", bytes.NewBuffer(tc.reqBody)))
			}

			response := httptest.NewRecorder()

			for _, req := range requests {
				f.ServeHTTP(response, req.WithContext(ctx))
				assert.Equal(tc.expectedStatusCode, response.Code)
			}

		})
	}
}

func TestSuccessfulAdd(t *testing.T) {
	var (
		logger = logging.NewTestLogger(nil, t)
		ctx    = logging.WithLogger(context.Background(), logger)
	)

	tests := []struct {
		description        string
		request            *http.Request
		newKey             bool
		expectedStatusCode int
		allowedFilters     FilterSet
		allowedFiltersSet  bool
	}{
		{
			description:        "Successful POST",
			request:            httptest.NewRequest("POST", "/", bytes.NewBuffer([]byte(`{"key": "test", "values": ["test1", "test2"]}`))).WithContext(ctx),
			newKey:             true,
			expectedStatusCode: http.StatusCreated,
		},
		{
			description:        "Successful POST Update",
			request:            httptest.NewRequest("POST", "/", bytes.NewBuffer([]byte(`{"key": "test", "values": ["random new value"]}`))).WithContext(ctx),
			expectedStatusCode: http.StatusOK,
		},
		{
			description:        "Successful PUT",
			request:            httptest.NewRequest("PUT", "/", bytes.NewBuffer([]byte(`{"key": "test", "values": ["test1", "test2"]}`))).WithContext(ctx),
			newKey:             true,
			expectedStatusCode: http.StatusCreated,
		},
		{
			description:        "Successful PUT Update",
			request:            httptest.NewRequest("PUT", "/", bytes.NewBuffer([]byte(`{"key": "test", "values": ["random new value"]}`))).WithContext(ctx),
			expectedStatusCode: http.StatusOK,
		},
		{
			description:        "Successful with Allowed Filters",
			request:            httptest.NewRequest("POST", "/", bytes.NewBuffer([]byte(`{"key": "test", "values": ["test1", "test2"]}`))).WithContext(ctx),
			newKey:             true,
			expectedStatusCode: http.StatusCreated,
			allowedFilters:     FilterSet(map[interface{}]bool{"test": true}),
			allowedFiltersSet:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			mockDeviceGate := new(mockDeviceGate)
			f := FilterHandler{
				Gate: mockDeviceGate,
			}

			mockDeviceGate.On("GetAllowedFilters").Return(tc.allowedFilters, tc.allowedFiltersSet).Once()
			mockDeviceGate.On("SetFilter", mock.AnythingOfType("string"), mock.Anything).Return(nil, tc.newKey).Once()
			mockDeviceGate.On("VisitAll", mock.Anything).Return(0).Once()

			response := httptest.NewRecorder()
			f.ServeHTTP(response, tc.request)
			assert.Equal(tc.expectedStatusCode, response.Code)
			assert.NotEmpty(response.Result().Body)
		})
	}

}

func TestSuccessfulDelete(t *testing.T) {
	var (
		logger   = logging.NewTestLogger(nil, t)
		ctx      = logging.WithLogger(context.Background(), logger)
		assert   = assert.New(t)
		response = httptest.NewRecorder()

		mockDeviceGate = new(mockDeviceGate)
		f              = FilterHandler{
			Gate: mockDeviceGate,
		}
	)

	req := httptest.NewRequest("DELETE", "/", bytes.NewBuffer([]byte(`{"key": "test"}`)))
	mockDeviceGate.On("DeleteFilter", "test").Return(true).Once()
	mockDeviceGate.On("VisitAll", mock.Anything).Return(0).Once()

	f.ServeHTTP(response, req.WithContext(ctx))
	assert.Equal(http.StatusOK, response.Code)
}

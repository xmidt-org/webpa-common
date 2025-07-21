// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package devicegate

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/sallust"
)

func TestServeHTTPGet(t *testing.T) {
	var (
		assert = assert.New(t)
		logger = sallust.Default()
		ctx    = sallust.With(context.Background(), logger)

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "/", nil)

		mockDeviceGate = new(mockDeviceGate)

		f = FilterHandler{
			Gate: mockDeviceGate,
		}
	)

	// nolint: typecheck
	mockDeviceGate.On("VisitAll", mock.Anything).Return(0)
	// nolint: typecheck
	mockDeviceGate.On("MarshalJSON").Return([]byte(`{}`), nil).Once()
	f.GetFilters(response, request.WithContext(ctx))
	assert.Equal(http.StatusOK, response.Code)
	assert.NotEmpty(response.Body)

}

func TestBadRequest(t *testing.T) {
	var (
		logger = sallust.Default()
		ctx    = sallust.With(context.Background(), logger)

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

	// nolint: typecheck
	mockDeviceGate.On("GetAllowedFilters").Return(&FilterSet{}, true)

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
				f.UpdateFilters(response, req.WithContext(ctx))
				assert.Equal(tc.expectedStatusCode, response.Code)
			}

		})
	}
}

func TestSuccessfulAdd(t *testing.T) {
	var (
		logger = sallust.Default()
		ctx    = sallust.With(context.Background(), logger)
	)

	tests := []struct {
		description        string
		request            *http.Request
		newKey             bool
		expectedStatusCode int
		allowedFilters     *FilterSet
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
			allowedFilters:     &FilterSet{Set: map[interface{}]bool{"test": true}},
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

			// nolint: typecheck
			mockDeviceGate.On("MarshalJSON").Return([]byte(`{}`), nil)
			// nolint: typecheck
			mockDeviceGate.On("GetAllowedFilters").Return(tc.allowedFilters, tc.allowedFiltersSet).Once()
			// nolint: typecheck
			mockDeviceGate.On("SetFilter", mock.AnythingOfType("string"), mock.Anything).Return(nil, tc.newKey).Once()
			// nolint: typecheck
			mockDeviceGate.On("VisitAll", mock.Anything).Return(0).Once()

			response := httptest.NewRecorder()
			f.UpdateFilters(response, tc.request)
			assert.Equal(tc.expectedStatusCode, response.Code)

		})
	}

}

func TestDelete(t *testing.T) {
	var (
		logger   = sallust.Default()
		ctx      = sallust.With(context.Background(), logger)
		assert   = assert.New(t)
		response = httptest.NewRecorder()

		mockDeviceGate = new(mockDeviceGate)
		f              = FilterHandler{
			Gate: mockDeviceGate,
		}
	)

	// nolint: typecheck
	mockDeviceGate.On("DeleteFilter", "test").Return(true).Once()
	// nolint: typecheck
	mockDeviceGate.On("VisitAll", mock.Anything).Return(0).Once()
	// nolint: typecheck
	mockDeviceGate.On("MarshalJSON").Return([]byte(`{}`), nil).Once()

	req := httptest.NewRequest("DELETE", "/", bytes.NewBuffer([]byte(`{"key": "test"}`)))
	f.DeleteFilter(response, req.WithContext(ctx))
	assert.Equal(http.StatusOK, response.Code)
}

func TestGateLogger(t *testing.T) {

	var (
		logger = sallust.Default()
		gate   = &FilterGate{
			FilterStore: FilterStore(map[string]Set{
				"partner-id": &FilterSet{
					Set: map[interface{}]bool{"comcast": true},
				},
			}),
		}

		gl = GateLogger{
			Logger: logger,
		}

		assert = assert.New(t)
	)

	tests := []struct {
		description   string
		next          http.Handler
		expectedEmpty bool
	}{
		{
			description: "Success",
			next: http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				response.WriteHeader(201)
				newCtx := context.WithValue(request.Context(), gateKey, gate)
				*request = *request.WithContext(newCtx)
			}),
		},
		{
			description: "No gate set in context",
			next: http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				response.WriteHeader(201)
			}),
			expectedEmpty: true,
		},
	}

	for _, tc := range tests {
		response := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		handler := gl.LogFilters(tc.next)
		handler.ServeHTTP(response, req)

		if tc.expectedEmpty {
			assert.Empty(response.Body)
		} else {
			assert.NotEmpty(response.Body)
		}

	}

}

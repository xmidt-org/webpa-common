// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package xhttp

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testConstant(t *testing.T, c Constant) {
	var (
		assert   = assert.New(t)
		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "/", nil)
	)

	c.ServeHTTP(response, request)
	assert.Equal(c.Code, response.Code)
	// nolint: typecheck
	if c.Header != nil {
		assert.Equal(c.Header, response.Header())
	} else {
		assert.Empty(response.Header())
	}

	assert.Equal(c.Body, response.Body.Bytes())
}

func TestConstant(t *testing.T) {
	testData := []Constant{
		{Code: http.StatusNotFound},
		// nolint: typecheck
		{Code: http.StatusServiceUnavailable, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: []byte(`{"message": "oh noes!"}`)},
	}

	for i, c := range testData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			testConstant(t, c)
		})
	}
}

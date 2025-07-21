// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package xhttp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testStaticHeaders(t *testing.T, extra, expected http.Header) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "/", nil)

		decoratedCalled = false
		next            = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			decoratedCalled = true
			assert.Equal(len(expected), len(response.Header()))
			for k, v := range expected {
				assert.Equal(v, response.Header()[k])
			}
		})

		constructor = StaticHeaders(extra)
	)

	require.NotNil(constructor)
	decorated := constructor(next)

	decorated.ServeHTTP(response, request)
	assert.True(decoratedCalled)
}

func TestStaticHeaders(t *testing.T) {
	t.Run("Nil", func(t *testing.T) { testStaticHeaders(t, nil, nil) })
	t.Run("Empty", func(t *testing.T) { testStaticHeaders(t, http.Header{}, nil) })
	t.Run("Several", func(t *testing.T) {
		testStaticHeaders(
			t,
			// nolint: typecheck
			http.Header{
				"Content-Type": {"application/json"},
				"x-something":  {"value1", "value2"},
				"eMPtY":        {},
			},
			// nolint: typecheck
			http.Header{
				"Content-Type": {"application/json"},
				"X-Something":  {"value1", "value2"},
				"Empty":        {},
			},
		)
	})
}

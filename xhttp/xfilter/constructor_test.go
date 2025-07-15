// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package xfilter

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithFilters(t *testing.T) {
	var (
		assert = assert.New(t)
		c      = new(constructor)
	)

	WithFilters()(c)
	assert.Empty(c.filters)

	WithFilters(Func(func(*http.Request) error { return nil }))(c)
	assert.Len(c.filters, 1)
}

func TestWithErrorEncoder(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		customCalled = false
		custom       = func(context.Context, error, http.ResponseWriter) {
			customCalled = true
		}

		c = new(constructor)
	)

	WithErrorEncoder(nil)(c)
	assert.NotNil(c.errorEncoder)

	WithErrorEncoder(custom)(c)
	require.NotNil(c.errorEncoder)

	c.errorEncoder(context.Background(), errors.New("expected"), httptest.NewRecorder())
	assert.True(customCalled)
}

func testNewConstructorDefault(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		delegate = http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			response.WriteHeader(599)
		})

		c = NewConstructor()

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "/", nil)
	)

	require.NotNil(c)
	decorated := c(delegate)
	require.NotNil(decorated)

	decorated.ServeHTTP(response, request)
	assert.Equal(599, response.Code)
}

func testNewConstructorFiltered(t *testing.T) {
	for _, i := range []int{1, 2, 5} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)

				expectedErr = errors.New("expected")
				response    = httptest.NewRecorder()
				request     = httptest.NewRequest("GET", "/", nil)
				delegate    = http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
					response.WriteHeader(299)
				})

				errorEncoder = func(_ context.Context, actualErr error, response http.ResponseWriter) {
					assert.Equal(expectedErr, actualErr)
					response.WriteHeader(599)
				}

				failAt        = i / 2
				filtersCalled int
				f             []Interface
			)

			for n := 0; n < i; n++ {
				if n == failAt {
					f = append(f, Func(func(*http.Request) error { filtersCalled++; return expectedErr }))
				} else {
					f = append(f, Func(func(*http.Request) error { filtersCalled++; return nil }))
				}
			}

			c := NewConstructor(WithErrorEncoder(errorEncoder), WithFilters(f...))
			require.NotNil(c)
			decorated := c(delegate)
			require.NotNil(decorated)
			decorated.ServeHTTP(response, request)

			assert.Equal(599, response.Code)
			assert.Equal(failAt+1, filtersCalled)
		})
	}
}

func testNewConstructorAllFiltersPass(t *testing.T) {
	for _, i := range []int{1, 2, 5} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)

				response = httptest.NewRecorder()
				request  = httptest.NewRequest("GET", "/", nil)
				delegate = http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
					response.WriteHeader(299)
				})

				filtersCalled int
				f             []Interface
			)

			for n := 0; n < i; n++ {
				f = append(f, Func(func(*http.Request) error { filtersCalled++; return nil }))
			}

			c := NewConstructor(WithFilters(f...))
			require.NotNil(c)
			decorated := c(delegate)
			require.NotNil(decorated)
			decorated.ServeHTTP(response, request)

			assert.Equal(299, response.Code)
			assert.Equal(i, filtersCalled)
		})
	}
}

func TestNewConstructor(t *testing.T) {
	t.Run("Default", testNewConstructorDefault)
	t.Run("Filtered", testNewConstructorFiltered)
	t.Run("AllFiltersPass", testNewConstructorAllFiltersPass)
}

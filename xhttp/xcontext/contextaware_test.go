// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package xcontext

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/justinas/alice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/sallust"
)

func TestSetContext(t *testing.T) {
	assert := assert.New(t)

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer, request = WithContext(writer, request, request.Context())

		assert.Panics(func() {
			SetContext(writer, nil)
		})
		writer = SetContext(writer, context.WithValue(writer.(ContextAware).Context(), "key", "value"))
		assert.Equal("value", writer.(ContextAware).Context().Value("key"))
		writer.WriteHeader(200)
		writer.Write([]byte("Hello World"))

	}))
	defer server.Close()

	r, err := http.NewRequest("GET", server.URL, nil)
	assert.NoError(err)
	r = r.WithContext(sallust.With(r.Context(), sallust.Default()))
	response, err := (&http.Client{}).Do(r)
	assert.NoError(err)
	assert.NotNil(response)
}

func TestSingleHandler(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer, request = WithContext(writer, request, request.Context())
		require.NotNil(writer)
		require.NotNil(request)

		writer.WriteHeader(200)
		writer.Write([]byte("Hello World"))

	}))
	defer server.Close()

	r, err := http.NewRequest("GET", server.URL, nil)
	assert.NoError(err)
	r = r.WithContext(sallust.With(r.Context(), sallust.Default()))
	response, err := (&http.Client{}).Do(r)
	assert.NoError(err)
	assert.NotNil(response)
}

func TestChain(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	body := "Hello World"
	bodyKey := "body"

	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer, request = WithContext(writer, request, request.Context())
		require.NotNil(writer)
		require.NotNil(request)

		writer.WriteHeader(200)
		writer.Write([]byte("Hello World"))

		if writer, ok := writer.(ContextAware); ok {
			writer.SetContext(context.WithValue(writer.Context(), bodyKey, body))
		} else {
			assert.Fail("Writer must be ContextAware")
		}
	})

	chain := alice.New(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := Context(w, r)
			w, r = WithContext(w, r, ctx)
			next.ServeHTTP(w, r)
			if writer, ok := w.(ContextAware); ok {
				assert.Equal(body, writer.Context().Value(bodyKey))
			} else {
				assert.Fail("Writer must be ContextAware")
			}
		})
	})

	server := httptest.NewServer(chain.Then(handler))
	defer server.Close()

	r, err := http.NewRequest("GET", server.URL, nil)
	assert.NoError(err)
	r = r.WithContext(sallust.With(r.Context(), sallust.Default()))
	response, err := (&http.Client{}).Do(r)
	assert.NoError(err)
	assert.NotNil(response)
}

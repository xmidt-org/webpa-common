package gate

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testNewConstructorNilGate(t *testing.T) {
	assert := assert.New(t)
	assert.Panics(func() {
		NewConstructor(nil, http.HandlerFunc(defaultClosedHandler))
	})
}

func testNewConstructorNilClosed(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		g       = New()

		nextCalled = false
		next       = http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			nextCalled = true
		})
	)

	constructor := NewConstructor(g, nil)
	require.NotNil(constructor)

	gated := constructor(next)
	require.NotNil(constructor)

	require.True(g.IsOpen())
	response := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/", nil)
	gated.ServeHTTP(response, request)
	assert.True(nextCalled)

	g.Lower()
	require.False(g.IsOpen())
	nextCalled = false
	response = httptest.NewRecorder()
	request = httptest.NewRequest("GET", "/", nil)
	gated.ServeHTTP(response, request)
	assert.False(nextCalled)
	assert.Equal(http.StatusServiceUnavailable, response.Code)
}

func testNewConstructorCustomClosed(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		g       = New()

		closedCalled = false
		closed       = http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			closedCalled = true
			response.WriteHeader(599)
		})

		nextCalled = false
		next       = http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			nextCalled = true
		})
	)

	constructor := NewConstructor(g, closed)
	require.NotNil(constructor)

	gated := constructor(next)
	require.NotNil(constructor)

	require.True(g.IsOpen())
	response := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/", nil)
	gated.ServeHTTP(response, request)
	assert.False(closedCalled)
	assert.True(nextCalled)

	g.Lower()
	require.False(g.IsOpen())
	closedCalled = false
	nextCalled = false
	response = httptest.NewRecorder()
	request = httptest.NewRequest("GET", "/", nil)
	gated.ServeHTTP(response, request)
	assert.True(closedCalled)
	assert.False(nextCalled)
	assert.Equal(599, response.Code)
}

func TestNewConstructor(t *testing.T) {
	t.Run("NilGate", testNewConstructorNilGate)
	t.Run("NilClosed", testNewConstructorNilClosed)
	t.Run("CustomClosed", testNewConstructorCustomClosed)
}

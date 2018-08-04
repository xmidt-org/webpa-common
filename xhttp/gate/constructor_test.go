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
		NewConstructor(nil)
	})
}

func testNewConstructorDefault(t *testing.T, c func(http.Handler) http.Handler, g Interface) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		next = http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			response.WriteHeader(201)
		})
	)

	require.NotNil(c)
	require.NotNil(g)

	decorated := c(next)
	require.NotNil(decorated)

	response := httptest.NewRecorder()
	decorated.ServeHTTP(response, httptest.NewRequest("GET", "/", nil))
	assert.Equal(201, response.Code)

	g.Lower()
	response = httptest.NewRecorder()
	decorated.ServeHTTP(response, httptest.NewRequest("GET", "/", nil))
	assert.Equal(http.StatusServiceUnavailable, response.Code)
}

func testNewConstructorCustomClosed(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		closed = http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			response.Header().Set("X-Test", "foobar")
			response.WriteHeader(599)
		})

		next = http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			response.WriteHeader(201)
		})

		g = New(true)
		c = NewConstructor(g, WithClosedHandler(closed))
	)

	require.NotNil(c)

	decorated := c(next)
	require.NotNil(decorated)

	response := httptest.NewRecorder()
	decorated.ServeHTTP(response, httptest.NewRequest("GET", "/", nil))
	assert.Equal(201, response.Code)
	assert.Empty(response.Header())

	g.Lower()
	response = httptest.NewRecorder()
	decorated.ServeHTTP(response, httptest.NewRequest("GET", "/", nil))
	assert.Equal(599, response.Code)
	assert.Equal("foobar", response.Header().Get("X-Test"))
}

func TestNewConstructor(t *testing.T) {
	t.Run("NilGate", testNewConstructorNilGate)
	t.Run("Default", func(t *testing.T) {
		g := New(true)
		testNewConstructorDefault(t, NewConstructor(g), g)

		g = New(true)
		testNewConstructorDefault(t, NewConstructor(g, WithClosedHandler(nil)), g)
	})

	t.Run("CustomClosed", testNewConstructorCustomClosed)
}

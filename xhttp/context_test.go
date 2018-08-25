package xhttp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	gokithttp "github.com/go-kit/kit/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testGetErrorEncoderDefault(t *testing.T) {
	assert := assert.New(t)
	assert.NotNil(GetErrorEncoder(context.Background()))
}

func testGetErrorEncoderCustom(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedCalled                        = false
		expected       gokithttp.ErrorEncoder = func(_ context.Context, _ error, _ http.ResponseWriter) {
			expectedCalled = true
		}

		actual = GetErrorEncoder(
			context.WithValue(context.Background(), errorEncoderKey{}, expected),
		)
	)

	require.NotNil(actual)
	actual(context.Background(), errors.New("expected"), httptest.NewRecorder())
	assert.True(expectedCalled)
}

func TestGetErrorEncoder(t *testing.T) {
	t.Run("Default", testGetErrorEncoderDefault)
	t.Run("Custom", testGetErrorEncoderCustom)
}

func testWithErrorEncoderDefault(t *testing.T) {
	var (
		assert = assert.New(t)
		ctx    = WithErrorEncoder(context.Background(), nil)
	)

	assert.Equal(context.Background(), ctx)
}

func testWithErrorEncoderCustom(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedCalled                        = false
		expected       gokithttp.ErrorEncoder = func(_ context.Context, _ error, _ http.ResponseWriter) {
			expectedCalled = true
		}

		ctx = WithErrorEncoder(context.Background(), expected)
	)

	require.NotNil(ctx)
	actual, ok := ctx.Value(errorEncoderKey{}).(gokithttp.ErrorEncoder)
	require.True(ok)
	require.NotNil(actual)

	actual(context.Background(), errors.New("expected"), httptest.NewRecorder())
	assert.True(expectedCalled)
}

func TestWithErrorEncoder(t *testing.T) {
	t.Run("Default", testWithErrorEncoderDefault)
	t.Run("Custom", testWithErrorEncoderCustom)
}

func testGetClientDefault(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(http.DefaultClient, GetClient(context.Background()))
}

func testGetClientCustom(t *testing.T) {
	var (
		assert = assert.New(t)

		expected = new(http.Client)
		actual   = GetClient(
			context.WithValue(context.Background(), httpClientKey{}, expected),
		)
	)

	assert.Equal(expected, actual)
}

func TestGetClient(t *testing.T) {
	t.Run("Default", testGetClientDefault)
	t.Run("Custom", testGetClientCustom)
}

func testWithClientDefault(t *testing.T) {
	var (
		assert = assert.New(t)
		ctx    = WithClient(context.Background(), nil)
	)

	assert.Equal(context.Background(), ctx)
}

func testWithClientCustom(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expected = new(http.Client)
		ctx      = WithClient(context.Background(), expected)
	)

	require.NotNil(ctx)
	actual, ok := ctx.Value(httpClientKey{}).(Client)
	require.True(ok)
	require.NotNil(actual)

	assert.Equal(expected, actual)
}

func TestWithClient(t *testing.T) {
	t.Run("Default", testWithClientDefault)
	t.Run("Custom", testWithClientCustom)
}

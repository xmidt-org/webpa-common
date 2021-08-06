package xcontext

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	gokithttp "github.com/go-kit/kit/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/webpa-common/v2/xhttp"
)

func testSetErrorEncoderDefault(t *testing.T) {
	var (
		assert = assert.New(t)
		ctx    = SetErrorEncoder(nil)(context.Background(), httptest.NewRequest("GET", "/", nil))
	)

	assert.NotNil(xhttp.GetErrorEncoder(ctx))
}

func testSetErrorEncoderCustom(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedCalled                        = false
		expected       gokithttp.ErrorEncoder = func(context.Context, error, http.ResponseWriter) {
			expectedCalled = true
		}

		actual = xhttp.GetErrorEncoder(
			SetErrorEncoder(expected)(context.Background(), httptest.NewRequest("GET", "/", nil)),
		)
	)

	require.NotNil(actual)
	actual(context.Background(), errors.New("expected"), httptest.NewRecorder())
	assert.True(expectedCalled)
}

func TestSetErrorEncoder(t *testing.T) {
	t.Run("Default", testSetErrorEncoderDefault)
	t.Run("Custom", testSetErrorEncoderCustom)
}

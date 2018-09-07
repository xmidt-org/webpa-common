package xfilter

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllow(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		f = Allow()
	)

	require.NotNil(f)
	assert.NoError(f.Allow(new(http.Request)))
}

func testRejectNil(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		f = Reject(nil)
	)

	require.NotNil(f)
	assert.NoError(f.Allow(new(http.Request)))
}

func testRejectNonNil(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedErr = errors.New("expected")
		f           = Reject(expectedErr)
	)

	require.NotNil(f)
	assert.Equal(expectedErr, f.Allow(new(http.Request)))
}

func TestReject(t *testing.T) {
	t.Run("Nil", testRejectNil)
	t.Run("NonNil", testRejectNonNil)
}

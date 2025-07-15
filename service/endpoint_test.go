// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/webpa-common/v2/service/accessor"
)

func testNewAccessorEndpointNilAccessor(t *testing.T) {
	assert := assert.New(t)
	assert.Panics(func() {
		NewAccessorEndpoint(nil)
	})
}

func testNewAccessorEndpointSuccess(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedKey = StringKey("expected key")

		a = new(accessor.MockAccessor)
		e = NewAccessorEndpoint(a)
	)

	require.NotNil(e)
	a.On("Get", []byte("expected key")).Return("expected instance", error(nil)).Once()

	response, err := e(context.Background(), expectedKey)
	assert.NoError(err)
	assert.Equal("expected instance", response)

	a.AssertExpectations(t)
}

func testNewAccessorEndpointError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedKey   = StringKey("expected key")
		expectedError = errors.New("expected error")

		a = new(accessor.MockAccessor)
		e = NewAccessorEndpoint(a)
	)

	require.NotNil(e)
	a.On("Get", []byte("expected key")).Return("", expectedError).Once()

	response, actualError := e(context.Background(), expectedKey)
	assert.Equal(expectedError, actualError)
	assert.Empty(response)

	a.AssertExpectations(t)
}

func TestNewAccessorEndpoint(t *testing.T) {
	t.Run("NilAccessor", testNewAccessorEndpointNilAccessor)
	t.Run("Success", testNewAccessorEndpointSuccess)
	t.Run("Error", testNewAccessorEndpointError)
}

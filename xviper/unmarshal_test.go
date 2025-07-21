// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package xviper

import (
	"errors"
	"testing"

	"github.com/spf13/viper"

	"github.com/stretchr/testify/assert"
)

func TestInvalidUnmarshaler(t *testing.T) {
	var (
		assert        = assert.New(t)
		expectedError = errors.New("expected unmarshal error")
	)

	assert.NoError(InvalidUnmarshaler{}.Unmarshal(nil))
	assert.Equal(
		expectedError,
		InvalidUnmarshaler{expectedError}.Unmarshal(nil),
	)
}

func testMustUnmarshalSuccess(t *testing.T) {
	var (
		assert      = assert.New(t)
		unmarshaler = new(mockUnmarshaler)
	)

	unmarshaler.On("Unmarshal", "valid", []viper.DecoderConfigOption(nil)).Return(nil).Once()

	assert.NotPanics(func() {
		MustUnmarshal(unmarshaler, "valid")
	})

	unmarshaler.AssertExpectations(t)
}

func testMustUnmarshalError(t *testing.T) {
	var (
		assert      = assert.New(t)
		unmarshaler = new(mockUnmarshaler)
	)

	unmarshaler.On("Unmarshal", "invalid", []viper.DecoderConfigOption(nil)).Return(errors.New("expected")).Once()

	assert.Panics(func() {
		MustUnmarshal(unmarshaler, "invalid")
	})

	unmarshaler.AssertExpectations(t)
}

func TestMustUnmarshal(t *testing.T) {
	t.Run("Success", testMustUnmarshalSuccess)
	t.Run("Error", testMustUnmarshalError)
}

func testMustKeyUnmarshalSuccess(t *testing.T) {
	var (
		assert      = assert.New(t)
		unmarshaler = new(mockKeyUnmarshaler)
	)

	unmarshaler.On("UnmarshalKey", "key", "value").Return(nil).Once()

	assert.NotPanics(func() {
		MustKeyUnmarshal(unmarshaler, "key", "value")
	})

	unmarshaler.AssertExpectations(t)
}

func testMustKeyUnmarshalError(t *testing.T) {
	var (
		assert      = assert.New(t)
		unmarshaler = new(mockKeyUnmarshaler)
	)

	unmarshaler.On("UnmarshalKey", "key", "value").Return(errors.New("expected")).Once()

	assert.Panics(func() {
		MustKeyUnmarshal(unmarshaler, "key", "value")
	})

	unmarshaler.AssertExpectations(t)
}

func TestMustKeyUnmarshal(t *testing.T) {
	t.Run("Success", testMustKeyUnmarshalSuccess)
	t.Run("Error", testMustKeyUnmarshalError)
}

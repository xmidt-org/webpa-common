package xviper

import (
	"errors"
	"testing"

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

func testUnmarshalSuccess(t *testing.T) {
	var (
		assert = assert.New(t)
		values = []interface{}{"one", "two", "three"}
	)

	for _, v := range [][]interface{}{values[0:1], values[0:2], values} {
		unmarshaler := new(mockUnmarshaler)

		for _, e := range v {
			unmarshaler.On("Unmarshal", e).Return(error(nil)).Once()
		}

		assert.NoError(Unmarshal(unmarshaler, v...))
		unmarshaler.AssertExpectations(t)
	}
}

func testUnmarshalError(t *testing.T) {
	var (
		assert        = assert.New(t)
		expectedError = errors.New("expected")
		values        = []interface{}{"one", "two", "three"}
	)

	for i := 0; i < len(values); i++ {
		unmarshaler := new(mockUnmarshaler)
		for j := 0; j < i; j++ {
			unmarshaler.On("Unmarshal", values[j]).Return(error(nil)).Once()
		}

		unmarshaler.On("Unmarshal", values[i]).Return(expectedError).Once()
		assert.Equal(expectedError, Unmarshal(unmarshaler, values...))
		unmarshaler.AssertExpectations(t)
	}
}

func TestUnmarshal(t *testing.T) {
	t.Run("Success", testUnmarshalSuccess)
	t.Run("Error", testUnmarshalError)
}

func TestMustUnmarshal(t *testing.T) {
	var (
		assert      = assert.New(t)
		unmarshaler = new(mockUnmarshaler)
	)

	unmarshaler.On("Unmarshal", "one").Return(error(nil)).Once()
	unmarshaler.On("Unmarshal", "two").Return(errors.New("expected")).Once()

	assert.Panics(func() {
		MustUnmarshal(unmarshaler, "one", "two")
	})

	unmarshaler.AssertExpectations(t)
}

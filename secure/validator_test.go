package secure

import (
	"errors"
	"github.com/Comcast/webpa-common/secure/key/keymock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewExactMatchValidator(t *testing.T) {
	assert := assert.New(t)

	token := &Token{
		tokenType: Basic,
		value:     "dGVzdDp0ZXN0Cg==",
	}

	successValidator := NewExactMatchValidator(token.value)
	assert.NotNil(successValidator)

	valid, err := successValidator.Validate(token)
	assert.True(valid)
	assert.Nil(err)

	failureValidator := NewExactMatchValidator("this should not be valid")
	assert.NotNil(failureValidator)

	valid, err = failureValidator.Validate(token)
	assert.False(valid)
	assert.Nil(err)
}

func TestNewJWSValidatorInvalidTokenType(t *testing.T) {
	assert := assert.New(t)

	mockJWSParser := &mockJWSParser{}
	mockResolver := &keymock.Resolver{}
	validator := &JWSValidator{
		Parser:   mockJWSParser,
		Resolver: mockResolver,
	}

	token := &Token{
		tokenType: Basic,
		value:     "does not matter",
	}

	valid, err := validator.Validate(token)
	assert.False(valid)
	assert.Nil(err)

	mockJWSParser.AssertExpectations(t)
	mockResolver.AssertExpectations(t)
}

func TestNewJWSValidatorInvalidJWT(t *testing.T) {
	assert := assert.New(t)

	mockJWSParser := &mockJWSParser{}
	mockResolver := &keymock.Resolver{}
	validator := &JWSValidator{
		Parser:   mockJWSParser,
		Resolver: mockResolver,
	}

	expectedError := errors.New("expected")
	token := &Token{
		tokenType: Bearer,
		value:     "",
	}

	mockJWSParser.On("ParseJWS", token).Return(nil, expectedError).Once()
	valid, err := validator.Validate(token)
	assert.False(valid)
	assert.Equal(expectedError, err)

	mockJWSParser.AssertExpectations(t)
	mockResolver.AssertExpectations(t)
}

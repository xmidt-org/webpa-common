package secure

import (
	"errors"
	"fmt"
	"github.com/Comcast/webpa-common/secure/key/keymock"
	"github.com/SermoDigital/jose"
	"github.com/SermoDigital/jose/jws"
	"github.com/SermoDigital/jose/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func ExampleSimpleJWSValidator() {
	// A basic validator with useful defaults
	// We need to use the publicKeyResolver, as that's what validates
	// the JWS signed with the private key
	validator := JWSValidator{
		Resolver: publicKeyResolver,
	}

	token := &Token{
		tokenType: Bearer,
		value:     string(testSerializedJWT),
	}

	valid, err := validator.Validate(token)
	fmt.Println(valid)
	fmt.Println(err)

	// Output:
	// true
	// <nil>
}

func TestValidatorFunc(t *testing.T) {
	assert := assert.New(t)
	expectedError := errors.New("expected")
	var validator Validator = ValidatorFunc(func(token *Token) (bool, error) { return false, expectedError })

	valid, err := validator.Validate(nil)
	assert.False(valid)
	assert.Equal(expectedError, err)
}

func TestExactMatchValidator(t *testing.T) {
	assert := assert.New(t)

	token := &Token{
		tokenType: Basic,
		value:     "dGVzdDp0ZXN0Cg==",
	}

	successValidator := ExactMatchValidator(token.value)
	assert.NotNil(successValidator)

	valid, err := successValidator.Validate(token)
	assert.True(valid)
	assert.Nil(err)

	failureValidator := ExactMatchValidator("this should not be valid")
	assert.NotNil(failureValidator)

	valid, err = failureValidator.Validate(token)
	assert.False(valid)
	assert.Nil(err)
}

func TestJWSValidatorInvalidTokenType(t *testing.T) {
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

func TestJWSValidatorInvalidJWT(t *testing.T) {
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

func TestJWSValidatorNoProtectedHeader(t *testing.T) {
	assert := assert.New(t)

	for _, empty := range []jose.Protected{nil, jose.Protected{}} {
		t.Logf("empty Protected header: %v", empty)
		token := &Token{tokenType: Bearer, value: "does not matter"}
		mockResolver := &keymock.Resolver{}

		mockJWS := &mockJWS{}
		mockJWS.On("Protected").Return(empty).Once()

		mockJWSParser := &mockJWSParser{}
		mockJWSParser.On("ParseJWS", token).Return(mockJWS, nil).Once()

		validator := &JWSValidator{
			Resolver: mockResolver,
			Parser:   mockJWSParser,
		}

		valid, err := validator.Validate(token)
		assert.False(valid)
		assert.Equal(err, ErrorNoProtectedHeader)

		mockResolver.AssertExpectations(t)
		mockJWS.AssertExpectations(t)
		mockJWSParser.AssertExpectations(t)
	}
}

func TestJWSValidatorNoSigningMethod(t *testing.T) {
	assert := assert.New(t)

	for _, badAlg := range []interface{}{nil, "", "this is not a valid signing method"} {
		t.Logf("badAlg: %v", badAlg)
		token := &Token{tokenType: Bearer, value: "does not matter"}
		mockResolver := &keymock.Resolver{}

		mockJWS := &mockJWS{}
		mockJWS.On("Protected").Return(jose.Protected{"alg": badAlg}).Once()

		mockJWSParser := &mockJWSParser{}
		mockJWSParser.On("ParseJWS", token).Return(mockJWS, nil).Once()

		validator := &JWSValidator{
			Resolver: mockResolver,
			Parser:   mockJWSParser,
		}

		valid, err := validator.Validate(token)
		assert.False(valid)
		assert.Equal(err, ErrorNoSigningMethod)

		mockResolver.AssertExpectations(t)
		mockJWS.AssertExpectations(t)
		mockJWSParser.AssertExpectations(t)
	}
}

// TestJWSValidatorResolverError also tests the correct key id determination
// when the header has a "kid" field vs the JWSValidator.DefaultKeyId member being set.
func TestJWSValidatorResolverError(t *testing.T) {
	assert := assert.New(t)

	var testData = []struct {
		headerKeyId   string
		defaultKeyId  string
		expectedKeyId string
	}{
		{"", "", ""},
		{"", "current", "current"},
		{"akey", "", "akey"},
		{"akey", "current", "akey"},
	}

	for _, record := range testData {
		t.Logf("%#v", record)
		token := &Token{tokenType: Bearer, value: "does not matter"}

		expectedResolverError := errors.New("expected resolver error")
		mockResolver := &keymock.Resolver{}
		mockResolver.On("ResolveKey", record.expectedKeyId).Return(nil, expectedResolverError).Once()

		mockJWS := &mockJWS{}
		mockJWS.On("Protected").Return(jose.Protected{"alg": "RS256", "kid": record.headerKeyId}).Once()

		mockJWSParser := &mockJWSParser{}
		mockJWSParser.On("ParseJWS", token).Return(mockJWS, nil).Once()

		validator := &JWSValidator{
			Resolver:     mockResolver,
			Parser:       mockJWSParser,
			DefaultKeyId: record.defaultKeyId,
		}

		valid, err := validator.Validate(token)
		assert.False(valid)
		assert.Equal(err, expectedResolverError)

		mockResolver.AssertExpectations(t)
		mockJWS.AssertExpectations(t)
		mockJWSParser.AssertExpectations(t)
	}
}

func TestJWSValidatorVerify(t *testing.T) {
	assert := assert.New(t)

	var testData = []struct {
		expectedValid       bool
		expectedVerifyError error
	}{
		{true, nil},
		{false, errors.New("expected Verify error")},
	}

	for _, record := range testData {
		t.Logf("%v", record)
		token := &Token{tokenType: Bearer, value: "does not matter"}

		expectedKey := interface{}(123)
		mockResolver := &keymock.Resolver{}
		mockResolver.On("ResolveKey", mock.AnythingOfType("string")).Return(expectedKey, nil).Once()

		expectedSigningMethod := jws.GetSigningMethod("RS256")
		assert.NotNil(expectedSigningMethod)

		mockJWS := &mockJWS{}
		mockJWS.On("Protected").Return(jose.Protected{"alg": "RS256"}).Once()
		mockJWS.On("Verify", expectedKey, expectedSigningMethod).Return(record.expectedVerifyError).Once()

		mockJWSParser := &mockJWSParser{}
		mockJWSParser.On("ParseJWS", token).Return(mockJWS, nil).Once()

		validator := &JWSValidator{
			Resolver: mockResolver,
			Parser:   mockJWSParser,
		}

		valid, err := validator.Validate(token)
		assert.Equal(record.expectedValid, valid)
		assert.Equal(record.expectedVerifyError, err)

		mockResolver.AssertExpectations(t)
		mockJWS.AssertExpectations(t)
		mockJWSParser.AssertExpectations(t)
	}
}

func TestJWSValidatorValidate(t *testing.T) {
	assert := assert.New(t)

	var testData = []struct {
		expectedValid         bool
		expectedValidateError error
		expectedJWTValidators []*jwt.Validator
	}{
		{true, nil, []*jwt.Validator{&jwt.Validator{}}},
		{true, nil, []*jwt.Validator{&jwt.Validator{}, &jwt.Validator{}}},
		{false, errors.New("expected Validate error 1"), []*jwt.Validator{&jwt.Validator{}}},
		{false, errors.New("expected Validate error 2"), []*jwt.Validator{&jwt.Validator{}, &jwt.Validator{}}},
	}

	for _, record := range testData {
		t.Logf("%v", record)
		token := &Token{tokenType: Bearer, value: "does not matter"}

		expectedKey := interface{}(123)
		mockResolver := &keymock.Resolver{}
		mockResolver.On("ResolveKey", mock.AnythingOfType("string")).Return(expectedKey, nil).Once()

		expectedSigningMethod := jws.GetSigningMethod("RS256")
		assert.NotNil(expectedSigningMethod)

		mockJWS := &mockJWS{}
		mockJWS.On("Protected").Return(jose.Protected{"alg": "RS256"}).Once()
		mockJWS.On("Validate", expectedKey, expectedSigningMethod, record.expectedJWTValidators).
			Return(record.expectedValidateError).
			Once()

		mockJWSParser := &mockJWSParser{}
		mockJWSParser.On("ParseJWS", token).Return(mockJWS, nil).Once()

		validator := &JWSValidator{
			Resolver:      mockResolver,
			Parser:        mockJWSParser,
			JWTValidators: record.expectedJWTValidators,
		}

		valid, err := validator.Validate(token)
		assert.Equal(record.expectedValid, valid)
		assert.Equal(record.expectedValidateError, err)

		mockResolver.AssertExpectations(t)
		mockJWS.AssertExpectations(t)
		mockJWSParser.AssertExpectations(t)
	}
}

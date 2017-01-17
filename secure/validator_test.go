package secure

import (
	"context"
	"errors"
	"fmt"
	"github.com/Comcast/webpa-common/secure/key"
	"github.com/SermoDigital/jose"
	"github.com/SermoDigital/jose/jws"
	"github.com/SermoDigital/jose/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
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

	valid, err := validator.Validate(nil, token)
	fmt.Println(valid, err)

	// Output:
	// true <nil>
}

func TestValidatorFunc(t *testing.T) {
	assert := assert.New(t)
	expectedError := errors.New("expected")
	var validator Validator = ValidatorFunc(func(ctx context.Context, token *Token) (bool, error) { return false, expectedError })

	valid, err := validator.Validate(nil, nil)
	assert.False(valid)
	assert.Equal(expectedError, err)
}

func TestValidators(t *testing.T) {
	assert := assert.New(t)
	var testData = [][]bool{
		[]bool{true},
		[]bool{false},
		[]bool{true, false},
		[]bool{false, true},
		[]bool{true, false, false},
		[]bool{false, true, false},
		[]bool{false, false, true},
	}

	for _, record := range testData {
		t.Logf("%v", record)
		token := &Token{}
		mocks := make([]interface{}, 0, len(record))
		validators := make(Validators, 0, len(record))

		// synthesize a chain of validators:
		// one mock for each entry.  until "true" is found,
		// validators should be called.  afterward, none
		// should be called.
		var (
			expectedValid bool
			expectedError error
		)

		for index, success := range record {
			mockValidator := &MockValidator{}
			mocks = append(mocks, mockValidator.Mock)
			validators = append(validators, mockValidator)

			if !expectedValid {
				expectedValid = success
				if success {
					expectedError = nil
				} else {
					expectedError = fmt.Errorf("expected validator error #%d", index)
				}

				mockValidator.On("Validate", nil, token).Return(expectedValid, expectedError).Once()
			}
		}

		valid, err := validators.Validate(nil, token)
		assert.Equal(expectedValid, valid)
		assert.Equal(expectedError, err)

		mock.AssertExpectationsForObjects(t, mocks...)
	}
}

func TestExactMatchValidator(t *testing.T) {
	assert := assert.New(t)

	token := &Token{
		tokenType: Basic,
		value:     "dGVzdDp0ZXN0Cg==",
	}

	successValidator := ExactMatchValidator(token.value)
	assert.NotNil(successValidator)

	valid, err := successValidator.Validate(nil, token)
	assert.True(valid)
	assert.Nil(err)

	failureValidator := ExactMatchValidator("this should not be valid")
	assert.NotNil(failureValidator)

	valid, err = failureValidator.Validate(nil, token)
	assert.False(valid)
	assert.Nil(err)
}

func TestJWSValidatorInvalidTokenType(t *testing.T) {
	assert := assert.New(t)

	mockJWSParser := &mockJWSParser{}
	mockResolver := &key.MockResolver{}
	validator := &JWSValidator{
		Parser:   mockJWSParser,
		Resolver: mockResolver,
	}

	token := &Token{
		tokenType: Basic,
		value:     "does not matter",
	}

	valid, err := validator.Validate(nil, token)
	assert.False(valid)
	assert.Nil(err)

	mockJWSParser.AssertExpectations(t)
	mockResolver.AssertExpectations(t)
}

func TestJWSValidatorInvalidJWT(t *testing.T) {
	assert := assert.New(t)

	mockJWSParser := &mockJWSParser{}
	mockResolver := &key.MockResolver{}
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
	valid, err := validator.Validate(nil, token)
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
		mockResolver := &key.MockResolver{}

		mockJWS := &mockJWS{}
		mockJWS.On("Protected").Return(empty).Once()

		mockJWSParser := &mockJWSParser{}
		mockJWSParser.On("ParseJWS", token).Return(mockJWS, nil).Once()

		validator := &JWSValidator{
			Resolver: mockResolver,
			Parser:   mockJWSParser,
		}

		valid, err := validator.Validate(nil, token)
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
		mockResolver := &key.MockResolver{}

		mockJWS := &mockJWS{}
		mockJWS.On("Protected").Return(jose.Protected{"alg": badAlg}).Once()

		mockJWSParser := &mockJWSParser{}
		mockJWSParser.On("ParseJWS", token).Return(mockJWS, nil).Once()

		validator := &JWSValidator{
			Resolver: mockResolver,
			Parser:   mockJWSParser,
		}

		valid, err := validator.Validate(nil, token)
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
		mockResolver := &key.MockResolver{}
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

		valid, err := validator.Validate(nil, token)
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

		mockPair := &key.MockPair{}
		expectedPublicKey := interface{}(123)
		mockPair.On("Public").Return(expectedPublicKey).Once()

		mockResolver := &key.MockResolver{}
		mockResolver.On("ResolveKey", mock.AnythingOfType("string")).Return(mockPair, nil).Once()

		expectedSigningMethod := jws.GetSigningMethod("RS256")
		assert.NotNil(expectedSigningMethod)

		mockJWS := &mockJWS{}
		mockJWS.On("Protected").Return(jose.Protected{"alg": "RS256"}).Once()
		mockJWS.On("Verify", expectedPublicKey, expectedSigningMethod).Return(record.expectedVerifyError).Once()

		claims := make(jws.Claims)
		claims.Set("capabilities", []string{"x1:webpa:api:.*:all"})
		mockJWS.On("Payload").Return(claims).Once()

		mockJWSParser := &mockJWSParser{}
		mockJWSParser.On("ParseJWS", token).Return(mockJWS, nil).Once()

		validator := &JWSValidator{
			Resolver: mockResolver,
			Parser:   mockJWSParser,
		}

		ctx := *new(context.Context)
		ctx = context.WithValue(ctx, "method", "post")
		ctx = context.WithValue(ctx, "path", "/api/foo/path")	

		valid, err := validator.Validate(ctx, token)
		assert.Equal(record.expectedValid, valid)
		assert.Equal(record.expectedVerifyError, err)

		mockPair.AssertExpectations(t)
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

		mockPair := &key.MockPair{}
		expectedPublicKey := interface{}(123)
		mockPair.On("Public").Return(expectedPublicKey).Once()

		mockResolver := &key.MockResolver{}
		mockResolver.On("ResolveKey", mock.AnythingOfType("string")).Return(mockPair, nil).Once()

		expectedSigningMethod := jws.GetSigningMethod("RS256")
		assert.NotNil(expectedSigningMethod)

		mockJWS := &mockJWS{}
		mockJWS.On("Protected").Return(jose.Protected{"alg": "RS256"}).Once()
		mockJWS.On("Validate", expectedPublicKey, expectedSigningMethod, record.expectedJWTValidators).
			Return(record.expectedValidateError).
			Once()

		claims := make(jws.Claims)
		claims.Set("capabilities", []string{"x1:webpa:api:.*:all"})
		mockJWS.On("Payload").Return(claims).Once()

		mockJWSParser := &mockJWSParser{}
		mockJWSParser.On("ParseJWS", token).Return(mockJWS, nil).Once()

		validator := &JWSValidator{
			Resolver:      mockResolver,
			Parser:        mockJWSParser,
			JWTValidators: record.expectedJWTValidators,
		}
		
		ctx := *new(context.Context)
		ctx = context.WithValue(ctx, "method", "post")
		ctx = context.WithValue(ctx, "path", "/api/foo/path")	

		valid, err := validator.Validate(ctx, token)
		assert.Equal(record.expectedValid, valid)
		assert.Equal(record.expectedValidateError, err)

		mockPair.AssertExpectations(t)
		mockResolver.AssertExpectations(t)
		mockJWS.AssertExpectations(t)
		mockJWSParser.AssertExpectations(t)
	}
}

func TestJWTValidatorFactory(t *testing.T) {
	assert := assert.New(t)
	now := time.Now().Unix()

	var testData = []struct {
		claims      jwt.Claims
		factory     JWTValidatorFactory
		expectValid bool
	}{
		{
			claims:      jwt.Claims{},
			factory:     JWTValidatorFactory{},
			expectValid: true,
		},
		{
			claims: jwt.Claims{
				"exp": now + 3600,
			},
			factory:     JWTValidatorFactory{},
			expectValid: true,
		},
		{
			claims: jwt.Claims{
				"exp": now - 3600,
			},
			factory:     JWTValidatorFactory{},
			expectValid: false,
		},
		{
			claims: jwt.Claims{
				"exp": now - 200,
			},
			factory: JWTValidatorFactory{
				ExpLeeway: 300,
			},
			expectValid: true,
		},
		{
			claims: jwt.Claims{
				"nbf": now + 3600,
			},
			factory:     JWTValidatorFactory{},
			expectValid: false,
		},
		{
			claims: jwt.Claims{
				"nbf": now - 3600,
			},
			factory:     JWTValidatorFactory{},
			expectValid: true,
		},
		{
			claims: jwt.Claims{
				"nbf": now + 200,
			},
			factory: JWTValidatorFactory{
				NbfLeeway: 300,
			},
			expectValid: true,
		},
	}

	for _, record := range testData {
		t.Logf("%#v", record)

		{
			t.Log("Simple case: no custom validate functions")
			validator := record.factory.New()
			assert.NotNil(validator)
			mockJWS := &mockJWS{}
			mockJWS.On("Claims").Return(record.claims).Once()

			err := validator.Validate(mockJWS)
			assert.Equal(record.expectValid, err == nil)

			mockJWS.AssertExpectations(t)
		}

		{
			for _, firstResult := range []error{nil, errors.New("first error")} {
				first := func(jwt.Claims) error {
					return firstResult
				}

				{
					t.Logf("One custom validate function returning: %v", firstResult)
					validator := record.factory.New(first)
					assert.NotNil(validator)
					mockJWS := &mockJWS{}
					mockJWS.On("Claims").Return(record.claims).Once()

					err := validator.Validate(mockJWS)
					assert.Equal(record.expectValid && firstResult == nil, err == nil)

					mockJWS.AssertExpectations(t)
				}

				for _, secondResult := range []error{nil, errors.New("second error")} {
					second := func(jwt.Claims) error {
						return secondResult
					}

					{
						t.Log("Two custom validate functions returning: %v, %v", firstResult, secondResult)
						validator := record.factory.New(first, second)
						assert.NotNil(validator)
						mockJWS := &mockJWS{}
						mockJWS.On("Claims").Return(record.claims).Once()

						err := validator.Validate(mockJWS)
						assert.Equal(
							record.expectValid && firstResult == nil && secondResult == nil,
							err == nil,
						)

						mockJWS.AssertExpectations(t)
					}
				}
			}
		}
	}
}

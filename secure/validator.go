package secure

import (
	"errors"
	"github.com/Comcast/webpa-common/store"
	"github.com/SermoDigital/jose/crypto"
	"github.com/SermoDigital/jose/jws"
	"github.com/SermoDigital/jose/jwt"
)

var (
	// DidNotMatch is returned by ExactMatch validators to indicate that
	// the token did not match the expected value.
	DidNotMatch = errors.New("The secure token did not match")
)

// Validator describes the behavior of a type which can validate tokens
type Validator interface {
	// Validate asserts that the given token is valid, most often verifying
	// the credentials in the token.  This method can return an arbitrary result
	// understood by the caller.
	Validate(*Token) (interface{}, error)
}

// ValidatorFunc is a function type that implements Validator
type ValidatorFunc func(*Token) (interface{}, error)

func (v ValidatorFunc) Validate(token *Token) (interface{}, error) {
	return v(token)
}

// ExactMatch produces a closure which validates that a token matches
// a given value exactly.  This validator simply returns the original
// token as the result along with a possible error indicating that the
// match was a failure.
func ExactMatch(match string) Validator {
	return ValidatorFunc(func(token *Token) (interface{}, error) {
		if match != token.value {
			return token, DidNotMatch
		}

		return token, nil
	})
}

// Verify returns a Validator closure that will verify JWT tokens using a given
// key, signing method, and JWT-specific validators.
func Verify(key store.Value, method crypto.SigningMethod, validators ...*jwt.Validator) Validator {
	return ValidatorFunc(func(token *Token) (interface{}, error) {
		jwt, err := jws.ParseJWT(token.Bytes())
		if err != nil {
			return nil, err
		}

		key, err := key.Load()
		if err != nil {
			return jwt, err
		}

		if err := jwt.Validate(key, method, validators...); err != nil {
			return jwt, err
		}

		return jwt, nil
	})
}

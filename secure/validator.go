package secure

import (
	"errors"
	"github.com/Comcast/webpa-common/secure/key"
	"github.com/SermoDigital/jose/jws"
	"github.com/SermoDigital/jose/jwt"
)

var (
	ErrorNoProtectedHeader = errors.New("Missing protected header")
	ErrorNoSigningMethod   = errors.New("Signing method (alg) is missing or unrecognized")
)

// Validator describes the behavior of a type which can validate tokens
type Validator interface {
	// Validate asserts that the given token is valid, most often verifying
	// the credentials in the token.  A separate error is returned to indicate
	// any problems during validation, such as the inability to access a network resource.
	// In general, the contract of this method is that a Token passes validation
	// if and only if it returns BOTH true and a nil error.
	Validate(*Token) (bool, error)
}

// ValidatorFunc is a function type that implements Validator
type ValidatorFunc func(*Token) (bool, error)

func (v ValidatorFunc) Validate(token *Token) (bool, error) {
	return v(token)
}

// NewExactMatchValidator produces a closure which validates that a token matches
// a given value exactly.  This type of validation is typically used with Basic
// authentication to match the user:password encoded string.  It can be used with
// any type of token, however.
func NewExactMatchValidator(match string) Validator {
	return ValidatorFunc(func(token *Token) (bool, error) {
		return match == token.value, nil
	})
}

// JWSValidator provides validation for JWT tokens encoded as JWS.
type JWSValidator struct {
	DefaultKeyId  string
	Resolver      key.Resolver
	Parser        JWSParser
	JWTValidators []*jwt.Validator
}

func (v *JWSValidator) Validate(token *Token) (valid bool, err error) {
	if token.Type() != Bearer {
		return
	}

	jwsToken, err := v.Parser.ParseJWS(token)
	if err != nil {
		return
	}

	protected := jwsToken.Protected()
	if len(protected) == 0 {
		err = ErrorNoProtectedHeader
		return
	}

	alg, _ := protected.Get("alg").(string)
	signingMethod := jws.GetSigningMethod(alg)
	if signingMethod == nil {
		err = ErrorNoSigningMethod
		return
	}

	keyId, _ := protected.Get("kid").(string)
	if len(keyId) == 0 {
		keyId = v.DefaultKeyId
	}

	key, err := v.Resolver.ResolveKey(keyId)
	if err != nil {
		return
	}

	if len(v.JWTValidators) > 0 {
		// all JWS implementations also implement jwt.JWT
		err = jwsToken.(jwt.JWT).Validate(key, signingMethod, v.JWTValidators...)
	} else {
		err = jwsToken.Verify(key, signingMethod)
	}

	valid = (err == nil)
	return
}

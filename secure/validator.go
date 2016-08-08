package secure

import (
	"fmt"
	"github.com/Comcast/webpa-common/secure/key"
	"github.com/SermoDigital/jose/jws"
	"github.com/SermoDigital/jose/jwt"
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

// ValidateExactMatch produces a closure which validates that a token matches
// a given value exactly.  This validator simply returns the original
// token as the result along with a possible error indicating that the
// match was a failure.
func ValidateExactMatch(match string) Validator {
	return ValidatorFunc(func(token *Token) (bool, error) {
		return match == token.value, nil
	})
}

// ValidateJWS produces a Validator for JWS tokens.  Comcast SAT tokens are JWS tokens.
func ValidateJWS(resolver key.Resolver, defaultKeyId string, jwtValidators ...*jwt.Validator) Validator {
	return ValidatorFunc(func(token *Token) (bool, error) {
		if token.Type() != Bearer {
			return false, nil
		}

		jwtToken, err := jws.ParseJWT(token.Bytes())
		if err != nil {
			return false, err
		}

		jwsToken, ok := jwtToken.(jws.JWS)
		if !ok {
			return false, fmt.Errorf("Token is not a valid JWS: %s", token)
		}

		header := jwsToken.Protected()
		signingMethodName, ok := header.Get("alg").(string)
		if !ok {
			return false, fmt.Errorf("Token does not define a signing method in the header: %s", token)
		}

		signingMethod := jws.GetSigningMethod(signingMethodName)
		if signingMethod == nil {
			return false, fmt.Errorf("Unknown signing method: %s", signingMethodName)
		}

		keyId, ok := header.Get("kid").(string)
		if len(keyId) == 0 {
			keyId = defaultKeyId
		}

		key, err := resolver.ResolveKey(keyId)
		if err != nil {
			return false, fmt.Errorf("Unable to resolve key with id: %s", keyId)
		}

		// Have to invoke Validate on the JWT type due to bad interface design in SermoDigital
		if err := jwtToken.Validate(key, signingMethod, jwtValidators...); err != nil {
			return false, err
		}

		return true, nil
	})
}

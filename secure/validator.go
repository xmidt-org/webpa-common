package secure

import (
	"fmt"
	"github.com/Comcast/webpa-common/secure/key"
	"github.com/SermoDigital/jose/crypto"
	"github.com/SermoDigital/jose/jws"
	"github.com/SermoDigital/jose/jwt"
)

const (
	// DefaultKeyId is the keyId used when no kid is present in the protected
	// header of a JWS
	DefaultKeyId = "current"
)

var (
	DefaultSigningMethod = jws.GetSigningMethod("RS256")
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
func ValidateJWS(resolver key.Resolver, jwtValidators ...*jwt.Validator) Validator {
	return ValidatorFunc(func(token *Token) (bool, error) {
		if token.Type() != Bearer {
			return false, nil
		}

		jwtToken, err := jws.ParseJWT(token.Bytes())
		if err != nil {
			return false, err
		}

		var (
			keyId         string
			signingMethod crypto.SigningMethod
		)

		// casting to a jws.JWS is the only way to get access to the protected header
		if jwsToken, ok := jwtToken.(jws.JWS); ok {
			// TODO: Support multiple protected headers?
			header := jwsToken.Protected()
			if alg, ok := header.Get("alg").(string); ok {
				signingMethod = jws.GetSigningMethod(alg)
			}

			if signingMethod == nil {
				signingMethod = DefaultSigningMethod
			}

			if keyId, _ = header.Get("kid").(string); len(keyId) == 0 {
				keyId = DefaultKeyId
			}
		} else {
			// we require a JWS token, since that is essentially a signed JWT
			return false, fmt.Errorf("Token is not a JWS: %s", token.String())
		}

		key, err := resolver.ResolveKey(keyId)
		if err != nil {
			return false, fmt.Errorf("Unable to resolve key with id: %s", keyId)
		}

		if err := jwtToken.Validate(key, signingMethod, jwtValidators...); err != nil {
			return false, err
		}

		return true, nil
	})
}

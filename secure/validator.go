package secure

import (
	"context"
	"errors"
	"fmt"
	"github.com/Comcast/webpa-common/secure/key"
	"github.com/SermoDigital/jose/jws"
	"github.com/SermoDigital/jose/jwt"
	"regexp"
	"strings"
	"time"
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
	Validate(context.Context, *Token) (bool, error, interface{})
}

// ValidatorFunc is a function type that implements Validator
type ValidatorFunc func(context.Context, *Token) (bool, error, interface{})

func (v ValidatorFunc) Validate(ctx context.Context, token *Token) (bool, error, interface{}) {
	return v(ctx, token)
}

// Validators is an aggregate Validator.  A Validators instance considers a token
// valid if any of its validators considers it valid.  An empty Validators rejects
// all tokens.
type Validators []Validator

func (v Validators) Validate(ctx context.Context, token *Token) (valid bool, err error, i interface{}) {
	for _, validator := range v {
		if valid, err, i = validator.Validate(ctx, token); valid && err == nil {
			return
		}
	}

	return
}

// ExactMatchValidator simply matches a token's value (exluding the prefix, such as "Basic"),
// to a string.
type ExactMatchValidator string

func (v ExactMatchValidator) Validate(ctx context.Context, token *Token) (bool, error, interface{}) {
	var i interface{}
	for _, value := range strings.Split(string(v), ",") {
		if value == token.value {
			return true, nil, i
		}
	}

	return false, nil, i
}

// JWSValidator provides validation for JWT tokens encoded as JWS.
type JWSValidator struct {
	DefaultKeyId  string
	Resolver      key.Resolver
	Parser        JWSParser
	JWTValidators []*jwt.Validator
}

// capabilityValidation determines if a claim's capability is valid
func capabilityValidation(ctx context.Context, capability string) (valid_capabilities bool) {
	pieces := strings.Split(capability, ":")

	if len(pieces) == 5 &&
		pieces[0] == "x1" &&
		pieces[1] == "webpa" {

		method_value, ok := ctx.Value("method").(string)
		if ok && (pieces[4] == "all" || strings.EqualFold(pieces[4], method_value)) {
			claimPath := fmt.Sprintf("/%s/[^/]+/%s", pieces[2], pieces[3])
			valid_capabilities, _ = regexp.MatchString(claimPath, ctx.Value("path").(string))
		}
	}

	return
}

func (v JWSValidator) Validate(ctx context.Context, token *Token) (valid bool, err error, pyld interface{}) {
	if token.Type() != Bearer {
		return
	}

	parser := v.Parser
	if parser == nil {
		parser = DefaultJWSParser
	}

	jwsToken, err := parser.ParseJWS(token)
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

	pair, err := v.Resolver.ResolveKey(keyId)
	if err != nil {
		return
	}

	// validate the signature
	if len(v.JWTValidators) > 0 {
		// all JWS implementations also implement jwt.JWT
		err = jwsToken.(jwt.JWT).Validate(pair.Public(), signingMethod, v.JWTValidators...)
	} else {
		err = jwsToken.Verify(pair.Public(), signingMethod)
	}

	if nil != err {
		return
	}

	// validate jwt token claims capabilities
	if caps, capOkay := jwsToken.Payload().(jws.Claims).Get("capabilities").([]interface{}); capOkay && len(caps) > 0 {

		/*  commenting out for now
		    1. remove code in use below
		    2. make sure to bring a back tests for this as well.
		        - TestJWSValidatorCapabilities()

				for c := 0; c < len(caps); c++ {
					if cap_value, ok := caps[c].(string); ok {
						if valid = capabilityValidation(ctx, cap_value); valid {
							return
						}
					}
				}
		*/
		// *****  REMOVE THIS CODE AFTER BRING BACK THE COMMENTED CODE ABOVE *****
		// ***** vvvvvvvvvvvvvvv *****
		return true, nil, jwsToken.Payload()
		// ***** ^^^^^^^^^^^^^^^ *****

	}

	// This fail
	return
}

// JWTValidatorFactory is a configurable factory for *jwt.Validator instances
type JWTValidatorFactory struct {
	Expected  jwt.Claims `json:"expected"`
	ExpLeeway int        `json:"expLeeway"`
	NbfLeeway int        `json:"nbfLeeway"`
}

func (f *JWTValidatorFactory) expLeeway() time.Duration {
	if f.ExpLeeway > 0 {
		return time.Duration(f.ExpLeeway) * time.Second
	}

	return 0
}

func (f *JWTValidatorFactory) nbfLeeway() time.Duration {
	if f.NbfLeeway > 0 {
		return time.Duration(f.NbfLeeway) * time.Second
	}

	return 0
}

// New returns a jwt.Validator using the configuration expected claims (if any)
// and a validator function that checks the exp and nbf claims.
//
// The SermoDigital library doesn't appear to do anything with the EXP and NBF
// members of jwt.Validator, but this Factory Method populates them anyway.
func (f *JWTValidatorFactory) New(custom ...jwt.ValidateFunc) *jwt.Validator {
	expLeeway := f.expLeeway()
	nbfLeeway := f.nbfLeeway()

	var validateFunc jwt.ValidateFunc
	customCount := len(custom)
	if customCount > 0 {
		validateFunc = func(claims jwt.Claims) (err error) {
			err = claims.Validate(time.Now(), expLeeway, nbfLeeway)
			for index := 0; index < customCount && err == nil; index++ {
				err = custom[index](claims)
			}

			return
		}
	} else {
		// if no custom validate functions were passed, use a simpler function
		validateFunc = func(claims jwt.Claims) error {
			return claims.Validate(time.Now(), expLeeway, nbfLeeway)
		}
	}

	return &jwt.Validator{
		Expected: f.Expected,
		EXP:      expLeeway,
		NBF:      nbfLeeway,
		Fn:       validateFunc,
	}
}

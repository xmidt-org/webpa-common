package handler

import (
	"github.com/Comcast/webpa-common/secure"
	"github.com/Comcast/webpa-common/secure/key"
	"github.com/Comcast/webpa-common/xmetrics"
	"github.com/SermoDigital/jose/jwt"
	"github.com/go-kit/kit/log"
	"github.com/justinas/alice"
	"github.com/spf13/viper"
)

var DefaultKeyID = "current" //todo: should this be provided by application code

type JWTValidator struct {
	// JWTKeys is used to create the key.Resolver for JWT verification keys
	Keys key.ResolverFactory

	// Custom is an optional configuration section that defines
	// custom rules for validation over and above the standard RFC rules.
	Custom secure.JWTValidatorFactory
}

//GetPreHandler returns a configured tunnel with requirements for requests to pass through before they reach some main handler
func GetPreHandler(v *viper.Viper, logger log.Logger, registry xmetrics.Registry) (preHandler *alice.Chain, err error) {
	m := secure.NewJWTValidationMeasures(registry)
	var validator secure.Validator

	if validator, err = getValidator(v, m); err == nil {

		authHandler := AuthorizationHandler{
			HeaderName:          "Authorization",
			ForbiddenStatusCode: 403,
			Validator:           validator,
			Logger:              logger,
		}

		authHandler.DefineMeasures(m)

		newPreHandler := alice.New(authHandler.Decorate)
		preHandler = &newPreHandler
	}
	return
}

//getValidator returns a validator for JWT/Basic tokens
//It reads in tokens from a config file. Zero or more tokens
//can be read.
func getValidator(v *viper.Viper, m *secure.JWTValidationMeasures) (validator secure.Validator, err error) {
	var jwtVals []JWTValidator

	v.UnmarshalKey("jwtValidators", &jwtVals)

	// if a JWTKeys section was supplied, configure a JWS validator
	// and append it to the chain of validators
	validators := make(secure.Validators, 0, len(jwtVals))

	for _, validatorDescriptor := range jwtVals {
		validatorDescriptor.Custom.defineMeasures(m)

		var keyResolver key.Resolver
		keyResolver, err = validatorDescriptor.Keys.NewResolver()
		if err != nil {
			validator = validators
			return
		}

		validator := secure.JWSValidator{
			DefaultKeyId:  DefaultKeyID,
			Resolver:      keyResolver,
			JWTValidators: []*jwt.Validator{validatorDescriptor.Custom.New()},
		}

		validator.defineMeasures(m)
		validators = append(validators, validator)
	}

	basicAuth := v.GetStringSlice("authHeader")
	for _, authValue := range basicAuth {
		validators = append(
			validators,
			secure.ExactMatchValidator(authValue),
		)
	}

	validator = validators

	return
}

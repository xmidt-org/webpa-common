package handler

import (
	"errors"
	"net/http"

	"github.com/SermoDigital/jose/jws"
	"github.com/xmidt-org/sallust"
	"github.com/xmidt-org/webpa-common/v2/secure"
	"github.com/xmidt-org/webpa-common/v2/xhttp"
	"go.uber.org/zap"
)

const (
	// The Content-Type value for JSON
	JsonContentType string = "application/json; charset=UTF-8"

	// The Content-Type header
	ContentTypeHeader string = "Content-Type"

	// The X-Content-Type-Options header
	ContentTypeOptionsHeader string = "X-Content-Type-Options"

	// NoSniff is the value used for content options for errors written by this package
	NoSniff string = "nosniff"
)

// AuthorizationHandler provides decoration for http.Handler instances and will
// ensure that requests pass the validator.  Note that secure.Validators is a Validator
// implementation that allows chaining validators together via logical OR.
type AuthorizationHandler struct {
	HeaderName          string
	ForbiddenStatusCode int
	Validator           secure.Validator
	Logger              *zap.Logger
	measures            *secure.JWTValidationMeasures
}

// headerName returns the authorization header to use, either a.HeaderName
// or secure.AuthorizationHeader if no header is supplied
func (a AuthorizationHandler) headerName() string {
	if len(a.HeaderName) > 0 {
		return a.HeaderName
	}

	return secure.AuthorizationHeader
}

// forbiddenStatusCode returns a.ForbiddenStatusCode if supplied, otherwise
// http.StatusForbidden is returned
func (a AuthorizationHandler) forbiddenStatusCode() int {
	if a.ForbiddenStatusCode > 0 {
		return a.ForbiddenStatusCode
	}

	return http.StatusForbidden
}

func (a AuthorizationHandler) logger() *zap.Logger {
	if a.Logger != nil {
		return a.Logger
	}

	return sallust.Default()
}

// Decorate provides an Alice-compatible constructor that validates requests
// using the configuration specified.
func (a AuthorizationHandler) Decorate(delegate http.Handler) http.Handler {
	// if there is no validator, there's no point in decorating anything
	if a.Validator == nil {
		return delegate
	}

	var (
		headerName          = a.headerName()
		forbiddenStatusCode = a.forbiddenStatusCode()
		logger              = a.logger()
	)

	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		headerValue := request.Header.Get(headerName)
		if len(headerValue) == 0 {
			logger.Info("missing header", zap.String("name", headerName))
			xhttp.WriteErrorf(response, forbiddenStatusCode, "missing header: %s", headerName)

			if a.measures != nil {
				a.measures.ValidationReason.With("reason", "missing_header").Add(1)
			}
			return
		}

		token, err := secure.ParseAuthorization(headerValue)
		if err != nil {
			logger.Error("invalid authorization header", zap.String("name", headerName), zap.Error(err))
			xhttp.WriteErrorf(response, forbiddenStatusCode, "Invalid authorization header [%s]: %s", headerName, err.Error())

			if a.measures != nil {
				a.measures.ValidationReason.With("reason", "invalid_header").Add(1)
			}
			return
		}

		contextValues := &ContextValues{
			Method: request.Method,
			Path:   request.URL.Path,
			Trust:  secure.Untrusted, // trust isn't set on the token until validation (ugh)
		}

		sharedContext := NewContextWithValue(request.Context(), contextValues)

		valid, err := a.Validator.Validate(sharedContext, token)
		if err == nil && valid {
			if err := populateContextValues(token, contextValues); err != nil {
				logger.Error("unable to populate context", zap.Error(err))
			}

			// this is absolutely horrible, but it's the only way we can do it for now.
			// TODO: address this in a redesign
			contextValues.Trust = token.Trust()
			delegate.ServeHTTP(response, request.WithContext(sharedContext))
			return
		}

		logger.Info(
			"request denied",
			zap.Bool("validator-response", valid),
			zap.NamedError("validator-error", err),
			zap.String("sat-client-id", contextValues.SatClientID),
			zap.String("method", request.Method),
			zap.Any("url", request.URL),
			zap.String("user-agent", request.Header.Get("User-Agent")),
			zap.Int64("content-length", request.ContentLength),
			zap.String("remoteAddress", request.RemoteAddr),
		)

		xhttp.WriteError(response, forbiddenStatusCode, "request denied")
	})
}

// DefineMeasures facilitates clients to define authHandler metrics tools
func (a *AuthorizationHandler) DefineMeasures(m *secure.JWTValidationMeasures) {
	a.measures = m
}

func populateContextValues(token *secure.Token, values *ContextValues) error {
	values.SatClientID = "N/A"

	if token.Type() != secure.Bearer {
		return nil
	}

	jwsToken, err := secure.DefaultJWSParser.ParseJWS(token)
	if err != nil {
		return err
	}

	claims, ok := jwsToken.Payload().(jws.Claims)
	if !ok {
		return errors.New("no claims")
	}

	if sub, ok := claims.Get("sub").(string); ok {
		values.SatClientID = sub
	}

	if allowedResources, ok := claims.Get("allowedResources").(map[string]interface{}); ok {
		if allowedPartners, ok := allowedResources["allowedPartners"].([]interface{}); ok {
			values.PartnerIDs = make([]string, 0, len(allowedPartners))
			for i := 0; i < len(allowedPartners); i++ {
				if value, ok := allowedPartners[i].(string); ok {
					values.PartnerIDs = append(values.PartnerIDs, value)
				}
			}
		}
	}

	return nil
}

package handler

import (
	"fmt"
	"net/http"

	"github.com/Comcast/webpa-common/contextcommon"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/secure"
	"github.com/SermoDigital/jose/jws"
	"github.com/go-kit/kit/log"
)

type contextKey struct{}

//handlerValuesKey is the key to set/get context values
var handlerValuesKey = contextKey{}

//ContextValues contains the values shared under the satClientIDKey from this package
type ContextValues struct {
	SatClientID string
	Method      string
	Path        string
}

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

// WriteJsonError writes a standard JSON error to the response
func WriteJsonError(response http.ResponseWriter, code int, message string) error {
	response.Header().Set(ContentTypeHeader, JsonContentType)
	response.Header().Set(ContentTypeOptionsHeader, NoSniff)

	response.WriteHeader(code)
	_, err := fmt.Fprintf(response, `{"message": "%s"}`, message)
	return err
}

// AuthorizationHandler provides decoration for http.Handler instances and will
// ensure that requests pass the validator.  Note that secure.Validators is a Validator
// implementation that allows chaining validators together via logical OR.
type AuthorizationHandler struct {
	HeaderName          string
	ForbiddenStatusCode int
	Validator           secure.Validator
	Logger              log.Logger
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

func (a AuthorizationHandler) logger() log.Logger {
	if a.Logger != nil {
		return a.Logger
	}

	return logging.DefaultLogger()
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
		errorLog            = logging.Error(logger)
	)

	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		headerValue := request.Header.Get(headerName)
		if len(headerValue) == 0 {
			errorLog.Log(logging.MessageKey(), "missing header", "name", headerName)
			WriteJsonError(response, forbiddenStatusCode, fmt.Sprintf("missing header: %s", headerName))
			return
		}

		token, err := secure.ParseAuthorization(headerValue)
		if err != nil {
			errorLog.Log(logging.MessageKey(), "invalid authorization header", "name", headerName, "token", headerValue, logging.ErrorKey(), err)
			WriteJsonError(response, forbiddenStatusCode, fmt.Sprintf("Invalid authorization header [%s]: %s", headerName, err.Error()))
			return
		}

		contextValues := &ContextValues{
			Method:      request.Method,
			Path:        request.URL.Path,
			SatClientID: extractSatClientID(token, logger),
		}

		sharedContext := contextcommon.NewContextWithValue(request.Context(), handlerValuesKey, contextValues)

		valid, err := a.Validator.Validate(sharedContext, token)
		if err == nil && valid {
			request = request.WithContext(sharedContext)
			// if any validator approves, stop and invoke the delegate
			delegate.ServeHTTP(response, request)
			return
		}

		errorLog.Log(
			logging.MessageKey(), "request denied",
			"validator-response", valid,
			"validator-error", err,
			"sat-client-id", contextValues.SatClientID,
			"token", headerValue,
			"method", request.Method,
			"url", request.URL,
			"user-agent", request.Header.Get("User-Agent"),
			"content-length", request.ContentLength,
			"remoteAddress", request.RemoteAddr,
		)

		WriteJsonError(response, forbiddenStatusCode, "request denied")
	})
}

func extractSatClientID(token *secure.Token, logger log.Logger) (satClientID string) {
	satClientID = "N/A"
	if token.Type() == secure.Bearer {
		if jwsObj, errJWSParse := secure.DefaultJWSParser.ParseJWS(token); errJWSParse == nil {
			if claims, ok := jwsObj.Payload().(jws.Claims); ok {
				if satClientIDStr, isString := claims.Get("sub").(string); isString {
					satClientID = satClientIDStr
				} else {
					logging.Error(logger).Log(logging.MessageKey(), "JWT Claim value was not of string type")
				}
			}
		} else {
			logging.Error(logger).Log(logging.MessageKey(), "Unexpected non-fatal JWS parse error", logging.ErrorKey(), errJWSParse)
		}
	}
	return
}

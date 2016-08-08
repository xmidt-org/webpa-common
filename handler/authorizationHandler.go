package handler

import (
	"fmt"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/secure"
	"net/http"
)

// AuthorizationHandler provides decoration for http.Handler instances and will
// ensure that requests pass one or more validators.
type AuthorizationHandler struct {
	HeaderName          string
	ForbiddenStatusCode int
	Validators          []secure.Validator
	Logger              logging.Logger
}

// Decorate provides an Alice-compatible constructor that validates requests
// using the configuration specified.
func (a AuthorizationHandler) Decorate(delegate http.Handler) http.Handler {
	// if there are no validators, there's no point in decorating anything
	if len(a.Validators) == 0 {
		return delegate
	}

	headerName := a.HeaderName
	if len(headerName) == 0 {
		headerName = secure.AuthorizationHeader
	}

	forbiddenStatusCode := a.ForbiddenStatusCode
	if forbiddenStatusCode < 100 {
		forbiddenStatusCode = http.StatusForbidden
	}

	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		headerValue := request.Header.Get(headerName)
		if len(headerValue) == 0 {
			message := fmt.Sprintf("No %s header", headerName)
			a.Logger.Error(message)
			WriteJsonError(response, forbiddenStatusCode, message)
			return
		}

		token, err := secure.ParseAuthorization(headerValue)
		if err != nil {
			message := fmt.Sprintf("Invalid authorization header [%s]: %s", headerValue, err.Error())
			a.Logger.Error(message)
			WriteJsonError(response, forbiddenStatusCode, message)
			return
		}

		for _, validator := range a.Validators {
			valid, err := validator.Validate(token)
			if err != nil {
				a.Logger.Error("Validation error: %s", err.Error())
			} else if valid {
				// if any validator approves, stop and invoke the delegate
				delegate.ServeHTTP(response, request)
				return
			}
		}

		a.Logger.Error("Request denied: %s", request)
		response.WriteHeader(forbiddenStatusCode)
	})
}

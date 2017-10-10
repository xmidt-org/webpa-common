package handler

import (
	"context"
	"fmt"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/secure"
	"github.com/SermoDigital/jose/jws"
	"net/http"
	"os"
)

// AuthorizationHandler provides decoration for http.Handler instances and will
// ensure that requests pass the validator.  Note that secure.Validators is a Validator
// implementation that allows chaining validators together via logical OR.
type AuthorizationHandler struct {
	HeaderName          string
	ForbiddenStatusCode int
	Validator           secure.Validator
	Logger              logging.Logger
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

func (a AuthorizationHandler) logger() logging.Logger {
	if a.Logger != nil {
		return a.Logger
	}

	return &logging.LoggerWriter{os.Stdout}
}

func obtainJwtSub(token interface{}) (sub string, ok bool) {
	if nil != token {
		if sub, ok = token.(jws.Claims).Subject(); ok {
			return
		}
	}

	return
}

// Decorate provides an Alice-compatible constructor that validates requests
// using the configuration specified.
func (a AuthorizationHandler) Decorate(delegate http.Handler) http.Handler {
	// if there is no validator, there's no point in decorating anything
	if a.Validator == nil {
		return delegate
	}

	headerName := a.headerName()
	forbiddenStatusCode := a.forbiddenStatusCode()
	logger := a.logger()

	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		headerValue := request.Header.Get(headerName)
		if len(headerValue) == 0 {
			message := fmt.Sprintf("No %s header", headerName)
			logger.Error(message)
			WriteJsonError(response, forbiddenStatusCode, message)
			return
		}

		token, err := secure.ParseAuthorization(headerValue)
		if err != nil {
			message := fmt.Sprintf("Invalid authorization header [%s]: %s", headerName, err.Error())
			logger.Error(message)
			WriteJsonError(response, forbiddenStatusCode, message)
			return
		}

		ctx := context.Background()
		ctx = context.WithValue(ctx, "method", request.Method)
		ctx = context.WithValue(ctx, "path", request.URL.Path)

		valid, err, pyld := a.Validator.Validate(ctx, token)

		if sub, subOk := obtainJwtSub(pyld); subOk {
			request.Header.Set("X-Webpa-Sat-Client-Id", sub)
		}

		if err != nil {
			logger.Error("Validation error: %s", err.Error())
		} else if valid {
			// if any validator approves, stop and invoke the delegate
			delegate.ServeHTTP(response, request)
			return
		}

		reqLogMsg := fmt.Sprintf("Request {Method: %s, URL: %s, User-Agent: %s, ContentLength: %d, RemoteAddr: %s}",
			request.Method, request.URL.String(), request.Header.Get("User-Agent"),
			request.ContentLength, request.RemoteAddr)

		logger.Error("Request denied: %s", reqLogMsg)
		response.WriteHeader(forbiddenStatusCode)
	})
}

package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/secure"
	"github.com/go-kit/kit/log"
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
		debugLog            = logging.Debug(logger)
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
			errorLog.Log(logging.MessageKey(), "invalid authorization header", "name", headerName, logging.ErrorKey(), err)
			WriteJsonError(response, forbiddenStatusCode, fmt.Sprintf("Invalid authorization header [%s]: %s", headerName, err.Error()))
			return
		}

		ctx := context.Background()
		ctx = context.WithValue(ctx, "method", request.Method)
		ctx = context.WithValue(ctx, "path", request.URL.Path)

		valid, err := a.Validator.Validate(ctx, token)
		if err != nil {
			errorLog.Log(logging.MessageKey(), "validation error", logging.ErrorKey(), err)
		} else if valid {
			// if any validator approves, stop and invoke the delegate
			delegate.ServeHTTP(response, request)
			return
		}

		errorLog.Log(
			logging.MessageKey(), "request denied",
			"method", request.Method,
			"url", request.URL,
			"user-agent", request.Header.Get("User-Agent"),
			"content-length", request.ContentLength,
			"remoteAddress", request.RemoteAddr,
		)

		//by this point, the token should be declared invalid
		debugLog.Log("invalid-token", headerValue)

		response.WriteHeader(forbiddenStatusCode)
	})
}

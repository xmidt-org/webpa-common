package handler

import (
	"context"
	"fmt"
	"github.com/Comcast/webpa-common/fact"
	"net/http"
)

// HttpError extends the error interface to include error information for HTTP responses
type HttpError interface {
	error
	Code() int
}

// httpError is the default, internal HttpError implementation
type httpError struct {
	code    int
	message string
}

func (err *httpError) Error() string {
	return err.message
}

func (err *httpError) Code() int {
	return err.code
}

// NewHttpError creates a new HttpError object.  This object implements
// go's builtin error interface.
func NewHttpError(code int, message string) HttpError {
	return &httpError{code, message}
}

// WriteJsonError writes a standard JSON error to the response
func WriteJsonError(response http.ResponseWriter, code int, message string) error {
	response.Header().Set(ContentTypeHeader, JsonContentType)
	response.Header().Set(ContentTypeOptionsHeader, NoSniff)

	response.WriteHeader(code)
	_, err := fmt.Fprintf(response, `{"message": "%s"}`, message)
	return err
}

// WriteError handles writing errors, possibly from panic, in a standard way.
// This method permits a variety of types for the err value.
func WriteError(response http.ResponseWriter, err interface{}) error {
	switch value := err.(type) {
	case HttpError:
		return WriteJsonError(response, value.Code(), value.Error())

	case error:
		return WriteJsonError(response, http.StatusInternalServerError, value.Error())

	case int:
		response.Header().Set(ContentTypeOptionsHeader, NoSniff)
		response.WriteHeader(value)

	case string:
		return WriteJsonError(response, http.StatusInternalServerError, value)

	case fmt.Stringer:
		return WriteJsonError(response, http.StatusInternalServerError, value.String())

	default:
		response.Header().Set(ContentTypeOptionsHeader, NoSniff)
		response.WriteHeader(http.StatusInternalServerError)
	}

	return nil
}

// Recover provides panic recovery for a chain of requests.  This function *must* be
// called as a deferred function.
func Recover(ctx context.Context, response http.ResponseWriter) {
	if recovered := recover(); recovered != nil {
		logger, ok := fact.Logger(ctx)
		if ok {
			logger.Error("Recovered: %v", recovered)
		}

		WriteError(response, recovered)
	}
}

package context

import (
	"fmt"
	"net/http"
	"strconv"
)

// HttpError extends the error interface to include error information for HTTP responses
type HttpError struct {
	code    int
	message string
}

func (err *HttpError) Error() string {
	return err.String()
}

func (err *HttpError) String() string {
	return strconv.Itoa(err.code) + ":" + err.message
}

func (err *HttpError) Code() int {
	return err.code
}

func (err *HttpError) Message() string {
	return err.message
}

// NewHttpError creates a new HttpError object.  This object implements
// go's builtin error interface.
func NewHttpError(code int, message string) *HttpError {
	return &HttpError{code, message}
}

// WriteJsonError writes a standard JSON error to the response
func WriteJsonError(response http.ResponseWriter, code int, message string) error {
	response.Header().Add(ContentTypeHeader, JsonContentType)
	response.WriteHeader(code)
	_, err := response.Write(
		[]byte(
			fmt.Sprintf(`{"message": "%s"}`, message),
		),
	)

	return err
}

// WriteError handles writing errors, possibly from panic, in a standard way.
// This method permits a variety of types for the err value.
func WriteError(response http.ResponseWriter, err interface{}) error {
	switch value := err.(type) {
	case HttpError:
		return WriteJsonError(response, value.code, value.message)

	case *HttpError:
		return WriteJsonError(response, value.code, value.message)

	case error:
		return WriteJsonError(response, http.StatusInternalServerError, value.Error())

	case int:
		response.WriteHeader(value)

	default:
		response.WriteHeader(http.StatusInternalServerError)
	}

	return nil
}

// RecoverError handles recovery during request processing.  This function must be
// invoked as a deferred function.  It handles writing an appropriate error response
// after a panic.
func RecoverError(logger Logger, response http.ResponseWriter) {
	if recovered := recover(); recovered != nil {
		if err := WriteError(response, recovered); err != nil {
			logger.Error("Unable to write error to response: %v", err)
		}
	}
}

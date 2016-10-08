package httperror

import (
	"fmt"
	"net/http"
)

const (
	// DefaultStatus is the response status code used when the error's code
	// is less than http.StatusBadRequest (400), i.e. when the code is not
	// an HTTP error code.
	DefaultStatus = http.StatusInternalServerError
)

// Interface represents an HTTP-specific error with additional metadata
// for the response.
type Interface interface {
	error
	String() string
	Status() int
	Header() http.Header
}

// httpError is the internal implementation of Interface
type httpError struct {
	message string
	status  int
	header  http.Header
}

func (err *httpError) Error() string {
	return err.message
}

func (err *httpError) String() string {
	return err.message
}

func (err *httpError) Status() int {
	return err.status
}

func (err *httpError) Header() http.Header {
	return err.header
}

// New returns an error containing the given internal metadata.  This constructor is appropriate
// for infrastructure that needs to return HTTP metadata about an error from code not directly
// part of an HTTP handler.
//
// For code that has access to the http.ResponseWriter, use WriteMessage instead
func New(message string, status int, header http.Header) Interface {
	if status < http.StatusBadRequest {
		status = DefaultStatus
	}

	return &httpError{
		message: message,
		status:  status,
		header:  header,
	}
}

// Write handles writing the given error to the response, taking care
// of the response status and any output headers.  This function can be used
// with errors other than HTTP errors.  It will provide default behavior in
// that case.
//
// If a status other than DefaultStatus is desired, use WriteMessage(response, err.Error(), DesiredStatus, nil)
// instead of this function.
func Write(response http.ResponseWriter, err error) (int, error) {
	if httpError, ok := err.(Interface); ok {
		return WriteMessage(response, httpError.Error(), httpError.Status(), httpError.Header())
	} else {
		return WriteMessage(response, err.Error(), DefaultStatus, nil)
	}
}

// WriteMessage handles writing full error message information out to a response.  This function
// avoids the overhead of creating a full blown HTTP error object.
//
// If status is not an HTTP error code, DefaultStatus is used.  The header is optional, and can be nil.
func WriteMessage(response http.ResponseWriter, message string, status int, header http.Header) (int, error) {
	if status < http.StatusBadRequest {
		status = DefaultStatus
	}

	for key, values := range header {
		for _, value := range values {
			response.Header().Add(key, value)
		}
	}

	response.Header().Set("Content-Type", "application/json; charset=UTF-8")
	response.Header().Set("X-Content-Type-Options", "nosniff")
	response.WriteHeader(status)

	return fmt.Fprintf(response, `{"message": "%s"}`, message)
}

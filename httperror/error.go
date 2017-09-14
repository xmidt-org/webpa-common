package httperror

import (
	"fmt"
	"net/http"

	gokithttp "github.com/go-kit/kit/transport/http"
)

// E is an HTTP-specific carrier of error information.  In addition to implementing error,
// this type also implements go-kit's StatusCoder and Headerer.
type E struct {
	Code   int
	Header http.Header
	Text   string
}

func (e *E) StatusCode() int {
	return e.Code
}

func (e *E) Headers() http.Header {
	return e.Header
}

func (e *E) Error() string {
	return e.Text
}

// StatusCode obtains a status code from an error if it implements StatusCoder.
// The second parameter is false if a code couldn't be retrieved.
func StatusCode(err error) (int, bool) {
	if coder, ok := err.(gokithttp.StatusCoder); ok {
		return coder.StatusCode(), true
	}

	return -1, false
}

// Header obtains an http.Header from an error if it implements Headerer.
// The second parameter is false if an http.Header couldn't be retrieved.
func Header(err error) (http.Header, bool) {
	if headerer, ok := err.(gokithttp.Headerer); ok {
		return headerer.Headers(), true
	}

	return nil, false
}

// Formatf provides printf-style functionality for writing out the results of some operation.
// The response status code is set to code, and a JSON message of the form {"code": %d, "message": "%s"} is
// written as the response body.  fmt.Sprintf is used to turn the format and parameters into a single string
// for the message.
//
// Although the typical use case for this function is to return a JSON error, this function can be used
// for non-error responses.
func Formatf(response http.ResponseWriter, code int, format string, parameters ...interface{}) (int, error) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(code)

	return fmt.Fprintf(
		response,
		`{"code": %d, "message": "%s"}`,
		code,
		fmt.Sprintf(format, parameters...),
	)
}

// Format provides print-style functionality for writing a JSON message as a response.  No format parameters
// are used.  The value parameter is subjected to the default stringizing rules of the fmt package.
func Format(response http.ResponseWriter, code int, value interface{}) (int, error) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(code)

	return fmt.Fprintf(
		response,
		`{"code": %d, "message": "%s"}`,
		code,
		value,
	)
}

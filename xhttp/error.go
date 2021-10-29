/**
 * Copyright 2021 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package xhttp

import (
	"fmt"
	"net/http"
)

// Error is an HTTP-specific carrier of error information.  In addition to implementing error,
// this type also implements go-kit's StatusCoder and Headerer.  The json.Marshaler interface
// is implemented so that the default go-kit error encoder will always emit a JSON message.
type Error struct {
	Code   int
	Header http.Header
	Text   string
}

func (e *Error) StatusCode() int {
	return e.Code
}

func (e *Error) Headers() http.Header {
	return e.Header
}

func (e *Error) Error() string {
	return e.Text
}

func (e *Error) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`{"code": %d, "text": "%s"}`, e.Code, e.Text)), nil
}

// WriteErrorf provides printf-style functionality for writing out the results of some operation.
// The response status code is set to code, and a JSON message of the form {"code": %d, "message": "%s"} is
// written as the response body.  fmt.Sprintf is used to turn the format and parameters into a single string
// for the message.
//
// Although the typical use case for this function is to return a JSON error, this function can be used
// for non-error responses.
func WriteErrorf(response http.ResponseWriter, code int, format string, parameters ...interface{}) (int, error) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(code)

	return fmt.Fprintf(
		response,
		`{"code": %d, "message": "%s"}`,
		code,
		fmt.Sprintf(format, parameters...),
	)
}

// WriteError provides print-style functionality for writing a JSON message as a response.  No format parameters
// are used.  The value parameter is subjected to the default stringizing rules of the fmt package.
func WriteError(response http.ResponseWriter, code int, value interface{}) (int, error) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(code)

	return fmt.Fprintf(
		response,
		`{"code": %d, "message": "%s"}`,
		code,
		value,
	)
}

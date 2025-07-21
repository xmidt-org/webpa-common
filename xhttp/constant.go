// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package xhttp

import "net/http"

// Constant represents an http.Handler that writes prebuilt, constant information to the response writer.
type Constant struct {
	Code   int
	Header http.Header
	Body   []byte
}

// ServeHTTP simply writes the configured information out to the response.
func (c Constant) ServeHTTP(response http.ResponseWriter, _ *http.Request) {
	for k, values := range c.Header {
		for _, v := range values {
			response.Header().Add(k, v)
		}
	}

	response.WriteHeader(c.Code)
	if len(c.Body) > 0 {
		response.Write(c.Body)
	}
}

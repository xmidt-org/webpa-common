// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package xcontext

import (
	"net/http"

	gokithttp "github.com/go-kit/kit/transport/http"
)

// Populate accepts any number of go-kit request functions and returns an Alice-style constructor that
// uses the request functions to build a context.  The resulting context is then assocated with the request
// prior to the next http.Handler being invoked.
//
// This function mimics the behavior of go-kit's transport/http package without requiring and endpoint with
// encoding and decoding.
func Populate(rf ...gokithttp.RequestFunc) func(http.Handler) http.Handler {
	if len(rf) > 0 {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				ctx := Context(response, request)
				for _, f := range rf {
					ctx = f(ctx, request)
				}

				response, request = WithContext(response, request, ctx)
				next.ServeHTTP(response, request)
			})
		}
	}

	return func(next http.Handler) http.Handler {
		return next
	}
}

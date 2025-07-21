// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package xhttp

import (
	"context"
	"net/http"

	gokithttp "github.com/go-kit/kit/transport/http"
)

type errorEncoderKey struct{}

// GetErrorEncoder returns the go-kit HTTP ErrorEncoder associated with the context.  If no
// encoder is present within the context, DefaultErrorEncoder is returned.
func GetErrorEncoder(ctx context.Context) gokithttp.ErrorEncoder {
	if ee, ok := ctx.Value(errorEncoderKey{}).(gokithttp.ErrorEncoder); ok {
		return ee
	}

	return gokithttp.DefaultErrorEncoder
}

// WithErrorEncoder associates a go-kit ErrorEncoder with the context.  If the supplied ErrorEncoder
// is nil, the supplied context is returned as is.
func WithErrorEncoder(ctx context.Context, ee gokithttp.ErrorEncoder) context.Context {
	if ee == nil {
		return ctx
	}

	return context.WithValue(ctx, errorEncoderKey{}, ee)
}

type httpClientKey struct{}

// GetClient returns the HTTP client associated with the context.  If no client is present
// in the context, http.DefaultClient is returned.
func GetClient(ctx context.Context) Client {
	if c, ok := ctx.Value(httpClientKey{}).(Client); ok {
		return c
	}

	return http.DefaultClient
}

// WithClient associates an HTTP Client with the context.  If the supplied client is
// nil, the supplied context is returned as is.
func WithClient(ctx context.Context, c Client) context.Context {
	if c == nil {
		return ctx
	}

	return context.WithValue(ctx, httpClientKey{}, c)
}

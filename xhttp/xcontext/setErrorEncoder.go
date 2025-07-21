// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package xcontext

import (
	"context"
	"net/http"

	gokithttp "github.com/go-kit/kit/transport/http"
	"github.com/xmidt-org/webpa-common/v2/xhttp"
)

// SetErrorEncoder is a ContextFunc strategy that injects a go-kit ErrorEncoder into the HTTP request context.
func SetErrorEncoder(ee gokithttp.ErrorEncoder) gokithttp.RequestFunc {
	return func(ctx context.Context, _ *http.Request) context.Context {
		return xhttp.WithErrorEncoder(ctx, ee)
	}
}

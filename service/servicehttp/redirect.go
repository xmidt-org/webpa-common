// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package servicehttp

import (
	"context"
	"net/http"
	"strings"

	gokithttp "github.com/go-kit/kit/transport/http"
	"github.com/xmidt-org/sallust"
	"go.uber.org/zap"
)

// Redirect returns a go-kit EncodeResponseFunc that redirects to the instance hashed by the accessor.
// If the original requestURI is populated under the go-kit key ContextKeyRequestURI, it is appended to
// the hashed instance.
func Redirect(redirectCode int) gokithttp.EncodeResponseFunc {
	if redirectCode < 300 {
		redirectCode = http.StatusTemporaryRedirect
	}

	return func(ctx context.Context, rw http.ResponseWriter, response interface{}) error {
		var (
			logger        = sallust.Get(ctx)
			instance      = response.(string)
			requestURI, _ = ctx.Value(gokithttp.ContextKeyRequestURI).(string)
		)

		if len(requestURI) > 0 {
			instance = instance + strings.TrimRight(requestURI, "/")
		}

		logger.Debug("redirecting", zap.String("instance", instance))
		rw.Header().Set("Location", instance)
		rw.WriteHeader(redirectCode)
		return nil
	}
}

// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package servicehttp

import (
	"net/http"
	"strings"

	"github.com/xmidt-org/sallust/sallusthttp"
	"github.com/xmidt-org/webpa-common/v2/service/accessor"
	"go.uber.org/zap"
)

// KeyFunc examines an HTTP request and produces the service key to use when finding
// an instance to use.
//
// The device.IDHashParser function is a valid KeyFunc, and is the typical one used by WebPA.
type KeyFunc func(*http.Request) ([]byte, error)

// RedirectHandler is an http.Handler that redirects all incoming requests using a key obtained
// from a request.  The Accessor is passed the key to return the appropriate instance to redirect to.
type RedirectHandler struct {
	// KeyFunc is the function used to extract a hash key from a request
	KeyFunc KeyFunc

	// Accessor produces instances given hash keys.  Note that a Subscription implements the Accessor interface.
	Accessor accessor.Accessor

	// RedirectCode is the HTTP status code sent as part of the redirect.  If not set, http.StatusTemporaryRedirect is used.
	RedirectCode int
}

func (rh *RedirectHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	key, err := rh.KeyFunc(request)
	ctxLogger := sallusthttp.Get(request)
	if err != nil {
		ctxLogger.Error("unable to obtain service key from request", zap.Error(err))
		http.Error(response, err.Error(), http.StatusBadRequest)
		return
	}

	instance, err := rh.Accessor.Get(key)
	if err != nil && instance == "" {
		ctxLogger.Error("accessor failed to return an instance", zap.Error(err))
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}

	instance += strings.TrimRight(request.RequestURI, "/")
	ctxLogger.Debug("redirecting", zap.String("instance", instance))

	code := rh.RedirectCode
	if code < 300 {
		code = http.StatusTemporaryRedirect
	}

	http.Redirect(response, request, instance, code)
}

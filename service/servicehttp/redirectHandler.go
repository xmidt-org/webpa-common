package servicehttp

import (
	"net/http"
	"strings"

	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/service"
	"github.com/go-kit/kit/log/level"
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
	Accessor service.Accessor

	// RedirectCode is the HTTP status code sent as part of the redirect.  If not set, http.StatusTemporaryRedirect is used.
	RedirectCode int
}

func (rh *RedirectHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	key, err := rh.KeyFunc(request)
	ctxLogger := logging.GetLogger(request.Context())
	if err != nil {
		ctxLogger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "unable to obtain service key from request", logging.ErrorKey(), err)
		http.Error(response, err.Error(), http.StatusBadRequest)
		return
	}

	instance, err := rh.Accessor.Get(key)
	if err != nil && instance == "" {
		ctxLogger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "accessor failed to return an instance", logging.ErrorKey(), err)
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}

	instance += strings.TrimRight(request.RequestURI, "/")
	ctxLogger.Log(level.Key(), level.DebugValue(), logging.MessageKey(), "redirecting", "instance", instance)

	code := rh.RedirectCode
	if code < 300 {
		code = http.StatusTemporaryRedirect
	}

	http.Redirect(response, request, instance, code)
}

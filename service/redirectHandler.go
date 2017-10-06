package service

import (
	"net/http"
	"strings"

	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/log"
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
	// Logger is the logger to which all output from ServeHTTP is sent
	Logger log.Logger

	// KeyFunc is the function used to extract a hash key from a request
	KeyFunc KeyFunc

	// Accessor produces instances given hash keys.  Note that a Subscription implements the Accessor interface.
	Accessor Accessor

	// RedirectCode is the HTTP status code sent as part of the redirect.  Normally clients set
	// this to http.StatusTemporaryRedirect.
	RedirectCode int
}

func (rh *RedirectHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	key, err := rh.KeyFunc(request)
	if err != nil {
		rh.Logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "unable to obtain service key from request", logging.ErrorKey(), err)
		http.Error(response, err.Error(), http.StatusBadRequest)
		return
	}

	instance, err := rh.Accessor.Get(key)
	if err != nil {
		rh.Logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "accessor failed to return an instance", logging.ErrorKey(), err)
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}

	instance += strings.TrimRight(request.URL.Path, "/") //keep original path with trailing '/' chars removed

	rh.Logger.Log(level.Key(), level.DebugValue(), logging.MessageKey(), "redirecting", "instance", instance)
	http.Redirect(response, request, instance, rh.RedirectCode)
}

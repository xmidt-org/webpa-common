package servicehttp

import (
	"net/http"
	"strings"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/logging/logginghttp"
	"github.com/Comcast/webpa-common/service"
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
	Accessor service.Accessor

	// RedirectCode is the HTTP status code sent as part of the redirect.  If not set, http.StatusTemporaryRedirect is used.
	RedirectCode int
}

func (rh *RedirectHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	key, err := rh.KeyFunc(request)

	if err != nil {
		rh.Logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "unable to obtain service key from request", logging.ErrorKey(), err)
		logHeaderFunc, ctx := logginghttp.SetLogger(rh.Logger, logginghttp.Header("X-WebPA-Device-Name", "device_id"), logginghttp.Header("Authorization", "authorization"))
		logHeaderFunc(request, ctx)
		http.Error(response, err.Error(), http.StatusBadRequest)
		return
	}

	instance, err := rh.Accessor.Get(key)
	if err != nil {
		rh.Logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "accessor failed to return an instance", logging.ErrorKey(), err)
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}

	instance += strings.TrimRight(request.RequestURI, "/")
	rh.Logger.Log(level.Key(), level.DebugValue(), logging.MessageKey(), "redirecting", "instance", instance)

	code := rh.RedirectCode
	if code < 300 {
		code = http.StatusTemporaryRedirect
	}

	http.Redirect(response, request, instance, code)
}

package service

import (
	"github.com/Comcast/webpa-common/logging"
	"net/http"
)

// NewRedirectHandler produces an http.Handler which simply redirects all requests
// using the results of an Accessor hash.  The supplied keyFunc is used to examine
// a request and return the []byte key.
func NewRedirectHandler(accessor Accessor, code int, keyFunc func(*http.Request) ([]byte, error), logger logging.Logger) http.Handler {
	if logger == nil {
		logger = logging.DefaultLogger()
	}

	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		key, err := keyFunc(request)
		if err != nil {
			logger.Error("Unable to obtain hash key from request: %s", err)
			http.Error(response, err.Error(), http.StatusBadRequest)
			return
		}

		node, err := accessor.Get(key)
		if err != nil {
			logger.Error("Accessor failed to return a node: %s", err)
			http.Error(response, err.Error(), http.StatusInternalServerError)
			return
		}

		redirectNode := ReplaceHostPort(node, request.URL)
		logger.Debug("Redirecting to: %s", redirectNode)
		http.Redirect(response, request, redirectNode, code)
	})
}

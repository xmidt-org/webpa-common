package service

// NewRedirectHandler produces an http.Handler which simply redirects all requests
// using the results of an Accessor hash.  The supplied keyFunc is used to examine
// a request and return the []byte key.
/*
func NewRedirectHandler(accessor Accessor, code int, keyFunc func(*http.Request) ([]byte, error), logger log.Logger) http.Handler {
	if logger == nil {
		logger = logging.DefaultLogger()
	}

	if code < 300 {
		code = http.StatusTemporaryRedirect
	}

	var (
		errorLog = logging.Error(logger)
		debugLog = logging.Debug(logger)
	)

	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		key, err := keyFunc(request)
		if err != nil {
			errorLog.Log(logging.MessageKey(), "Unable to obtain hash key from request", "error", err)
			http.Error(response, err.Error(), http.StatusBadRequest)
			return
		}

		node, err := accessor.Get(key)
		if err != nil {
			errorLog.Log(logging.MessageKey(), "Accessor failed to return a node", "error", err)
			http.Error(response, err.Error(), http.StatusInternalServerError)
			return
		}

		redirectNode := ReplaceHostPort(node, request.URL)
		debugLog.Log(logging.MessageKey(), "Redirecting", "node", redirectNode)
		http.Redirect(response, request, redirectNode, code)
	})
}
*/

package handler

import (
	"github.com/Comcast/webpa-common/context"
	"github.com/Comcast/webpa-common/logging"
	"net/http"
)

// recovery is the internal ChainHandler function.  For simplicity, clients must call Recovery()
// to obtain a reference to this function, which gives a little syntactic sugar.
func recovery(logger logging.Logger, response http.ResponseWriter, request *http.Request, next http.Handler) {
	defer func() {
		if recovered := recover(); recovered != nil {
			logger.Error("Recovering from panic: %v", recovered)
			context.WriteError(response, recovered)
		}
	}()

	next.ServeHTTP(response, request)
}

func Recovery() ChainHandler {
	return ChainHandlerFunc(recovery)
}

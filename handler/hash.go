package handler

import (
	"context"
	"fmt"
	"github.com/Comcast/webpa-common/fact"
	"github.com/Comcast/webpa-common/hash"
	"net/http"
)

// Hash returns a redirector using the default status code of http.StatusTemporaryRedirect.
func Hash(serviceHash hash.ServiceHash) ContextHandler {
	return HashCustom(serviceHash, http.StatusTemporaryRedirect)
}

// HashCustom provides a ContextHandler that redirects requests based on a ServiceHash.
// The context must have a device identifier, which is then used as the key supplied to the hash.
func HashCustom(serviceHash hash.ServiceHash, redirectCode int) ContextHandler {
	return ContextHandlerFunc(func(ctx context.Context, response http.ResponseWriter, request *http.Request) {
		address, err := serviceHash.Get(fact.MustDeviceId(ctx).Bytes())
		if err == nil {
			target := address + request.URL.Path
			http.Redirect(response, request, target, redirectCode)
		} else {
			message := fmt.Sprintf("No nodes available: %s", err.Error())
			if logger, ok := fact.Logger(ctx); ok {
				logger.Warn(message)
			}

			// service hash errors should be http.StatusServiceUnavailable, since
			// they almost always indicate that no nodes are in the hash due to no
			// available service nodes in the remote system (e.g. zookeeper)
			WriteJsonError(
				response,
				http.StatusServiceUnavailable,
				message,
			)
		}
	})
}

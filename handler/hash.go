package handler

import (
	"fmt"
	"github.com/Comcast/webpa-common/fact"
	"github.com/Comcast/webpa-common/hash"
	"golang.org/x/net/context"
	"net/http"
)

// Hash provides a ContextHandler that redirects requests based on a ServiceHash.
func Hash(serviceHash hash.ServiceHash, code int) ContextHandler {
	return ContextHandlerFunc(func(ctx context.Context, response http.ResponseWriter, request *http.Request) {
		address, err := serviceHash.Get(fact.MustDeviceId(ctx).Bytes())
		if err != nil {
			message := fmt.Sprintf("No nodes available: %s", err.Error())
			fact.MustLogger(ctx).Warn(message)

			// service hash errors should be http.StatusServiceUnavailable, since
			// they almost always indicate that no nodes are in the hash due to no
			// available service nodes in the remote system (e.g. zookeeper)
			WriteJsonError(
				response,
				http.StatusServiceUnavailable,
				message,
			)

			return
		}

		target := address + request.URL.Path
		http.Redirect(response, request, target, code)
	})
}

package handler

import (
	"fmt"
	"github.com/Comcast/webpa-common/context"
	"github.com/Comcast/webpa-common/hash"
	"net/http"
)

// HashRedirector provides a ContextHandler that redirects requests based on a ServiceHash.
func HashRedirector(serviceHash hash.ServiceHash) ContextHandler {
	return ContextHandlerFunc(func(requestContext context.Context, response http.ResponseWriter, request *http.Request) {
		address, err := serviceHash.Get(requestContext.DeviceId().Bytes())
		if err != nil {
			// service hash errors should be http.StatusServiceUnavailable, since
			// they almost always indicate that no nodes are in the hash due to no
			// available service nodes in the remote system (e.g. zookeeper)
			context.WriteJsonError(
				response,
				http.StatusServiceUnavailable,
				fmt.Sprintf("No nodes avaiable: %s", err.Error()),
			)

			return
		}

		target := address + request.URL.Path
		http.Redirect(response, request, target, http.StatusTemporaryRedirect)
	})
}

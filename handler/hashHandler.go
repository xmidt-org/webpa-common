package handler

import (
	"github.com/Comcast/webpa-common/context"
	"github.com/Comcast/webpa-common/hash"
	"net/http"
)

// HashRedirector provides a ContextHandler that redirects requests based on a ServiceHash.
func HashRedirector(serviceHash hash.ServiceHash) ContextHandler {
	return ContextHandlerFunc(func(requestContext context.Context, response http.ResponseWriter, request *http.Request) {
		address, err := serviceHash.Get(requestContext.DeviceId().Bytes())
		if err != nil {
			context.WriteError(response, err)
			return
		}
		
		target := address + request.URL.Path
		http.Redirect(response, request, target, http.StatusTemporaryRedirect)
	})
}

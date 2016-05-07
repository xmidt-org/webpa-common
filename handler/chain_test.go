package handler

import (
	"github.com/Comcast/webpa-common/context"
	"os"
	"net/http"
	"fmt"
)

func sampleContextHandler(requestContext context.Context, response http.ResponseWriter, request *http.Request) {
	response.WriteHeader(http.StatusContinue)
	response.Write([]byte("hello, world"))
}

type remoteSystem struct {
	
}

func (r remoteSystem) Connected() bool {
	return true
}

// ExampleBasicChain shows the typical usage pattern for chained handlers
func ExampleBasicChain() {
	logger := context.DefaultLogger{os.Stdout}
	handler := Chain{
		Recovery(),
		RemoteGate(remoteSystem{}, http.StatusServiceUnavailable, "Service Unavailable"),
	}.DecorateContext(logger, ContextHandlerFunc(sampleContextHandler))
	
	fmt.Println(handler)
}

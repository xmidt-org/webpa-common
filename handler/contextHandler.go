package handler

import (
	"github.com/Comcast/webpa-common/context"
	"github.com/Comcast/webpa-common/logging"
	"net/http"
)

// ContextHandler defines the behavior of types which can handle HTTP requests inside a WebPA context
type ContextHandler interface {
	// ServeHTTP handles an HTTP request within an enclosing WebPA context
	ServeHTTP(context.Context, http.ResponseWriter, *http.Request)
}

// ContextHandlerFunc is a function type that implements ContextHandler
type ContextHandlerFunc func(context.Context, http.ResponseWriter, *http.Request)

func (f ContextHandlerFunc) ServeHTTP(requestContext context.Context, response http.ResponseWriter, request *http.Request) {
	f(requestContext, response, request)
}

// contextHttpHandler is an internal type that adapts http.Handler onto ContextHandler
type contextHttpHandler struct {
	logger   logging.Logger
	delegate ContextHandler
}

func (c *contextHttpHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	requestContext, err := context.NewContext(c.logger, request)
	if err != nil {
		c.logger.Error("Unable to create WebPA context: %v", err)
		context.WriteError(response, err)
		return
	}

	c.delegate.ServeHTTP(requestContext, response, request)
}

// NewContextHttpHandler returns a new http.Handler that passes a distinct WebPA context object
// for each request to the delegate function.
func NewContextHttpHandler(logger logging.Logger, delegate ContextHandler) http.Handler {
	return &contextHttpHandler{
		logger:   logger,
		delegate: delegate,
	}
}

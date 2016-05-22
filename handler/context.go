package handler

import (
	"golang.org/x/net/context"
	"net/http"
)

// ContextHandler defines the behavior of types which can handle HTTP requests inside a WebPA context
type ContextHandler interface {
	// ServeHTTP handles an HTTP request within an enclosing WebPA context
	ServeHTTP(context.Context, http.ResponseWriter, *http.Request)
}

// ContextHandlerFunc is a function type that implements ContextHandler
type ContextHandlerFunc func(context.Context, http.ResponseWriter, *http.Request)

func (f ContextHandlerFunc) ServeHTTP(ctx context.Context, response http.ResponseWriter, request *http.Request) {
	f(ctx, response, request)
}

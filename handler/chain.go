package handler

import (
	"context"
	"net/http"
)

// ChainHandler represents an HTTP handler type that is one part of a chain of handlers.
type ChainHandler interface {
	ServeHTTP(context.Context, http.ResponseWriter, *http.Request, ContextHandler)
}

// ChainHandlerFunc is a function type that implements ChainHandler
type ChainHandlerFunc func(context.Context, http.ResponseWriter, *http.Request, ContextHandler)

func (f ChainHandlerFunc) ServeHTTP(requestContext context.Context, response http.ResponseWriter, request *http.Request, next ContextHandler) {
	f(requestContext, response, request, next)
}

// chainLink represents one node in a chain of handlers.  It is essentially
// a linked list node.
type chainLink struct {
	current ChainHandler
	next    ContextHandler
}

func (link *chainLink) ServeHTTP(requestContext context.Context, response http.ResponseWriter, request *http.Request) {
	link.current.ServeHTTP(requestContext, response, request, link.next)
}

// Chain represents an ordered slice of ChainHandlers that will be applied to each request.
type Chain []ChainHandler

// Len returns the number of handlers in this chain
func (chain Chain) Len() int {
	return len(chain)
}

// Decorate produces a single http.Handler that executes each handler in the chain in sequence
// before finally executing a ContextHandler.  The given Context is passed through the chain,
// and may be modified at each step.
func (chain Chain) Decorate(initial context.Context, contextHandler ContextHandler) http.Handler {
	var decorated ContextHandler = contextHandler

	// have to decorate in reverse order in order to guarantee that the decorators
	// execute in declared order.
	for index := len(chain) - 1; index >= 0; index-- {
		decorated = &chainLink{
			current: chain[index],
			next:    decorated,
		}
	}

	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		defer Recover(initial, response)
		decorated.ServeHTTP(initial, response, request)
	})
}

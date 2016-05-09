package handler

import (
	"github.com/Comcast/webpa-common/logging"
	"net/http"
)

// ChainHandler represents an HTTP handler type that is one part of a chain of handlers.
type ChainHandler interface {
	ServeHTTP(logging.Logger, http.ResponseWriter, *http.Request, http.Handler)
}

// ChainHandlerFunc is a function type that implements ChainHandler
type ChainHandlerFunc func(logging.Logger, http.ResponseWriter, *http.Request, http.Handler)

func (f ChainHandlerFunc) ServeHTTP(logger logging.Logger, response http.ResponseWriter, request *http.Request, next http.Handler) {
	f(logger, response, request, next)
}

// chainLink is an internal type that acts like one node in a linked list.
// This type maintains a reference to a ChainHandler and the next http.Handler,
// which in turn can be another chainedHandler.
type chainLink struct {
	handler ChainHandler
	logger  logging.Logger
	next    http.Handler
}

func (link *chainLink) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	link.handler.ServeHTTP(link.logger, response, request, link.next)
}

// Chain represents an ordered list of ChainHandlers that will decorate a given http.Handler.
type Chain []ChainHandler

// Decorate applies the chain of handlers to the given delegate.  The order in which each handler
// is executed is the same as the order within the Chain slice.
func (chain Chain) Decorate(logger logging.Logger, delegate http.Handler) http.Handler {
	decorated := delegate
	for index := len(chain) - 1; index >= 0; index-- {
		decorated = &chainLink{
			handler: chain[index],
			logger:  logger,
			next:    decorated,
		}
	}

	return decorated
}

// DecorateContext is a variant of Decorate that uses a ContextHandler as the delegate.
func (chain Chain) DecorateContext(logger logging.Logger, delegate ContextHandler) http.Handler {
	return chain.Decorate(logger, NewContextHttpHandler(logger, delegate))
}

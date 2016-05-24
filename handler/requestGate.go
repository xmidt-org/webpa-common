package handler

import (
	"github.com/Comcast/webpa-common/fact"
	"golang.org/x/net/context"
	"net/http"
)

// Connection represents some sort of subsystem or remote system that has a notion
// of being available
type Connection interface {
	// Connected returns a boolean indicating whether the abstract system
	// denoted by this instance is available.
	Connected() bool
}

// ConnectionFunc is a function type that implements Connection
type ConnectionFunc func() bool

func (f ConnectionFunc) Connected() bool {
	return f()
}

// MergeConnections returns an aggregate Connection object
// that returns false if any of the given Connection objects return false.
func MergeConnections(connections ...Connection) Connection {
	return ConnectionFunc(func() bool {
		for _, connection := range connections {
			if !connection.Connected() {
				return false
			}
		}

		return true
	})
}

// RequestGate returns a ChainHandler whose requests are gated by the given RequestGate
func RequestGate(connection Connection, unavailableStatus int, unavailableMessage string) ChainHandler {
	return ChainHandlerFunc(func(ctx context.Context, response http.ResponseWriter, request *http.Request, next ContextHandler) {
		if !connection.Connected() {
			fact.MustLogger(ctx).Error("Request denied: %s", unavailableMessage)

			WriteJsonError(
				response,
				unavailableStatus,
				unavailableMessage,
			)
		}

		next.ServeHTTP(ctx, response, request)
	})
}

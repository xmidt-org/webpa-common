package handler

import (
	"github.com/Comcast/webpa-common/context"
	"net/http"
)

// Remote represents a system that this application depends upon that may be disconnected due
// to errors.
type Remote interface {
	// Connected returns a boolean indicating whether this remote system is still connected
	Connected() bool
}

// Gate provides the standard behavior for gated requests.
type Gate interface {
	// ShouldRequestProceed determines if the given request should be allowed to continue.
	// The first return value is an error that describes why the the error should not proceed.
	// The second return value is a boolean indicating whether the request should proceed.
	// If the error is nil but the boolean is true, then this package denies the request with
	// a standard message.
	ShouldRequestProceed(*http.Request) (error, bool)
}

// GateFunc is a function type that implements RequestGate
type GateFunc func(*http.Request) (error, bool)

func (f GateFunc) ShouldRequestProceed(request *http.Request) (error, bool) {
	err, allow := f(request)
	return err, allow
}

// NewRemoteGate creates a Gate using a Remote.  This method is a factory for a Gate,
// while the similarly named RemoteGate() function is a factory for a ChainHandler that
// enforces a gate.
func NewRemoteGate(remote Remote, unavailableStatus int, unavailableMessage string) Gate {
	unavailableError := context.NewHttpError(unavailableStatus, unavailableMessage)
	return GateFunc(func(*http.Request) (error, bool) {
		connected := remote.Connected()
		if connected {
			return nil, true
		}

		return unavailableError, false
	})
}

// CheckRequest does the work of applying the RequestGate and determining if the request
// should proceed.  This function returns a boolean indicating whether the caller should
// continue with the request.  If this function returns false, then a response will have
// already been written.
func CheckRequest(gate Gate, response http.ResponseWriter, request *http.Request) bool {
	err, allow := gate.ShouldRequestProceed(request)
	if !allow {
		switch value := err.(type) {
		case context.HttpError:
			context.WriteJsonError(response, value.Code(), value.Error())

		case error:
			context.WriteJsonError(response, http.StatusServiceUnavailable, value.Error())

		default:
			context.WriteJsonError(response, http.StatusServiceUnavailable, "Service Unavailable")
		}
	}

	return allow
}

// RequestGate returns an http.Handler whose requests are gated by the given RequestGate
func RequestGate(gate Gate) ChainHandler {
	return ChainHandlerFunc(func(logger context.Logger, response http.ResponseWriter, request *http.Request, next http.Handler) {
		if !CheckRequest(gate, response, request) {
			logger.Warn("Request denied")
			return
		}

		next.ServeHTTP(response, request)
	})
}

// RemoteGate is a variant of RequestGate that operates in terms of a Remote with a static
// HTTP response that should be returned to callers should Remote.Connected() return false.
func RemoteGate(remote Remote, unavailableStatus int, unavailableMessage string) ChainHandler {
	gate := NewRemoteGate(remote, unavailableStatus, unavailableMessage)
	return RequestGate(gate)
}

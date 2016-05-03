package context

import (
	"net/http"
)

// RequestGate provides the standard behavior for gated requests.  Clients may supply
// an implementation of this interface.
type RequestGate interface {
	// ShouldRequestProceed determines if the given request should be allowed to continue.
	// The first return value is an error that describes why the the error should not proceed.
	// The second return value is a boolean indicating whether the request should proceed.
	// If the error is nil but the boolean is true, then this package denies the request with
	// a standard message.
	ShouldRequestProceed(*http.Request) (error, bool)
}

// RequestGateFunc is a function type that implements RequestGate
type RequestGateFunc func(*http.Request) (error, bool)

func (f RequestGateFunc) ShouldRequestProceed(request *http.Request) (error, bool) {
	err, allow := f(request)
	return err, allow
}

// AllowAll is a RequestGate that simply allows all requests.  Use this instead
// of passing a nil RequestGate.
func AllowAll(request *http.Request) (error, bool) {
	return nil, true
}

// CheckRequest does the work of applying the RequestGate and determining if the request
// should proceed.  This function returns a boolean indicating whether the caller should
// continue with the request.  If this function returns false, then a response will have
// already been written.
func CheckRequest(requestGate RequestGate, response http.ResponseWriter, request *http.Request) bool {
	err, allow := requestGate.ShouldRequestProceed(request)
	if !allow {
		switch value := err.(type) {
		case HttpError:
			WriteJsonError(response, value.code, value.message)

		case *HttpError:
			WriteJsonError(response, value.code, value.message)

		case error:
			WriteJsonError(response, http.StatusServiceUnavailable, error.Error())

		default:
			WriteJsonError(response, http.StatusServiceUnavailable, ServiceUnavailableMessage)
		}
	}

	return allow
}

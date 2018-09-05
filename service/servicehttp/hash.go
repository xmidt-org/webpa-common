package servicehttp

import (
	"net/http"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/service"
	"github.com/go-kit/kit/log/level"
)

// HashOptions are the configuration parameters for a Hash constructor.
type HashOptions struct {
	// Accessor is the required service Accessor used to hash request keys
	Accessor service.Accessor

	// KeyFunc is used to extract a key from a request.  device.IDHashParser is typically used,
	// although this field has no default.
	KeyFunc KeyFunc

	// RejectCode is the HTTP response code returned when a request does not hash to any of the Self values.
	// If unset (or less than 400), http.StatusCone is used by default.
	RejectCode int

	// ErrorCode is the HTTP response code returned when an error in hashing occurs.  If unset (or less than 400),
	// http.StatusInternalServerError is used by default.
	ErrorCode int

	// Self is the set of instances that refer to the enclosing server.  If empty, no check is performed.  However,
	// the hash is still executed as a verification that the hash does work against the key.
	Self []string
}

// Hash produces an Alice-style constructor that decorates http Handlers with hash enforcement:  requests that do not
// hash to a list of "self" instances are rejected.
func Hash(o HashOptions) func(http.Handler) http.Handler {
	self := make(map[string]bool, len(o.Self))
	for _, i := range o.Self {
		self[i] = true
	}

	if o.RejectCode < 400 {
		o.RejectCode = http.StatusGone
	}

	if o.ErrorCode < 400 {
		o.ErrorCode = http.StatusInternalServerError
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			key, err := o.KeyFunc(request)
			if err != nil {
				logging.GetLogger(request.Context()).Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "Unable to extract device key", logging.ErrorKey(), err)
				response.WriteHeader(o.ErrorCode)
				return
			}

			hashed, err := o.Accessor.Get(key)
			if err != nil {
				logging.GetLogger(request.Context()).Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "Unable to hash key", logging.ErrorKey(), err)
				response.WriteHeader(o.ErrorCode)
				return
			}

			if len(self) > 0 && !self[hashed] {
				logging.GetLogger(request.Context()).Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "Device does not hash to this server", "key", string(key))
				response.WriteHeader(o.RejectCode)
				return
			}

			next.ServeHTTP(response, request)
		})
	}
}

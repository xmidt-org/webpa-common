package servicehttp

import (
	"net/http"

	"github.com/Comcast/webpa-common/device"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/service"
	"github.com/Comcast/webpa-common/xhttp/xfilter"
	"github.com/go-kit/kit/log/level"
)

// NewHashFilter constructs an xfilter that enforces device hashing to an instance that represents this server process.
// Any request that does not pass the self predicate will be rejected with the reject error.
//
// The returned filter will check the request's context for a device id, using that to hash with if one is found.
// Otherwise, the device key is parsed from the request via device.IDHashParser.
func NewHashFilter(a service.Accessor, reject error, self func(string) bool) xfilter.Interface {
	// allow any nil parameter to simply disable the filtering
	if a == nil || reject == nil || self == nil {
		return xfilter.Allow()
	}

	return xfilter.Func(func(r *http.Request) error {
		var key []byte

		if id, ok := device.GetID(r.Context()); ok {
			key = id.Bytes()
		} else {
			var err error
			if key, err = device.IDHashParser(r); err != nil {
				return err
			}
		}

		i, err := a.Get(key)
		if err != nil {
			return err
		}

		if !self(i) {
			logging.GetLogger(r.Context()).Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "device does not hash to this instance", "hashKey", string(key), logging.ErrorKey(), reject, "instance", i)
			return reject
		}

		return nil
	})
}

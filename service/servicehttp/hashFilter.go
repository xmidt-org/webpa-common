package servicehttp

import (
	"net/http"
	"strings"

	"github.com/Comcast/webpa-common/device"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/service"
	"github.com/Comcast/webpa-common/xhttp/xfilter"
	"github.com/go-kit/kit/log/level"
)

// NewHashFilter constructs an xfilter that enforces device hashing to one or more "self" instances.
// Any request that does not hash to one of the "self" instances is rejected by returning the supplied error.
// If self is empty, an always-allow xfilter is returned instead.
//
// The returned filter will check the request's context for a device id, using that to hash with if one is found.
// Otherwise, the device key is parsed from the request via device.IDHashParser.
func NewHashFilter(a service.Accessor, reject error, self ...string) xfilter.Interface {
	// filter out any blank strings from the self, which allows for injected values that can
	// disable the hash filter.
	var filteredSelf []string
	for _, s := range self {
		s = strings.TrimSpace(s)
		if len(s) > 0 {
			filteredSelf = append(filteredSelf, s)
		}
	}

	if len(filteredSelf) == 0 {
		return xfilter.Allow()
	}

	// compute this value once, for logging
	selfValue := strings.Join(filteredSelf, ",")

	selfSet := make(map[string]bool, len(self))
	for _, i := range filteredSelf {
		selfSet[i] = true
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

		if !selfSet[i] {
			logging.GetLogger(r.Context()).Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "device does not hash to this instance", "key", string(key), logging.ErrorKey(), reject, "instance", i, "self", selfValue)
			return reject
		}

		return nil
	})
}

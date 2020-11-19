package devicegate

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/go-kit/kit/log/level"
	"github.com/xmidt-org/webpa-common/device"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/xhttp"
)

const (
	metadataMapLocation = "metadata_map"
	claimsLocation      = "claims"
)

var (
	emptyVal = struct{}{}
)

type FilterGate struct {
	filters        map[string]map[interface{}]struct{}
	allowedFilters map[string]struct{}
	lock           sync.RWMutex
}

type FilterGateOption func(*FilterGate)

type filterRequest struct {
	Key    string
	Values []interface{}
}

func New(options ...FilterGateOption) *FilterGate {
	fg := &FilterGate{
		filters:        make(map[string]map[interface{}]struct{}),
		allowedFilters: make(map[string]struct{}),
	}

	for _, o := range options {
		o(fg)
	}

	return fg

}

func WithAllowedFilters(allowedFilters map[string]struct{}) FilterGateOption {
	return func(fg *FilterGate) {
		if allowedFilters != nil {
			fg.allowedFilters = allowedFilters
		} else {
			fg.allowedFilters = make(map[string]struct{})
		}
	}
}

func (f *FilterGate) PrettifyFilters() map[string][]interface{} {
	copy := make(map[string][]interface{})

	f.lock.RLock()
	for k, v := range f.filters {
		var filters []interface{}
		for filter := range v {
			filters = append(filters, filter)
		}

		copy[k] = filters
	}
	f.lock.RUnlock()

	return copy
}

func (f *FilterGate) FiltersCopy() map[string]map[interface{}]struct{} {
	filtersCopy := make(map[string]map[interface{}]struct{})

	f.lock.RLock()
	for key, val := range f.filters {
		valCopy := make(map[interface{}]struct{})

		for k, v := range val {
			valCopy[k] = v
		}

		filtersCopy[key] = valCopy
	}
	f.lock.RUnlock()

	return filtersCopy
}

func (f *FilterGate) Filters() map[string]map[interface{}]struct{} {
	return f.filters
}

//allows for adding and removing filters
//Will completely overwrite current filters in place
func (f *FilterGate) EditFilters(key string, values []interface{}, add bool) bool {
	f.lock.Lock()
	defer f.lock.Unlock()
	if add {
		valuesMap := make(map[interface{}]struct{})

		for _, v := range values {
			valuesMap[v] = emptyVal
		}

		f.filters[key] = valuesMap

		return true
	} else {

		_, ok := f.filters[key]

		if ok {
			delete(f.filters, key)
			return true
		}

		return false
	}
}

func (f *FilterGate) AllowConnection(d device.Interface) (bool, device.MatchResult) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	for filterKey, filterValues := range f.filters {

		// check if filter is in claims
		if claimsMatch(filterKey, filterValues, d.Metadata()) {
			return false, device.MatchResult{
				Location: metadataMapLocation,
				Key:      filterKey,
			}
		}

		// check if filter is in metadata map
		if metadataMapMatch(filterKey, filterValues, d.Metadata()) {
			return false, device.MatchResult{
				Location: claimsLocation,
				Key:      filterKey,
			}
		}

	}

	return true, device.MatchResult{}
}

func metadataMapMatch(keyToCheck string, filterValues map[interface{}]struct{}, m *device.Metadata) bool {
	metadataVal := m.Load(keyToCheck)
	if metadataVal != nil {
		switch t := metadataVal.(type) {
		case interface{}:
			return filterMatch(filterValues, t)
		case []interface{}:
			return filterMatch(filterValues, t...)

		}
	}

	return false

}

func claimsMatch(keyToCheck string, filterValues map[interface{}]struct{}, m *device.Metadata) bool {
	claimsMap := m.Claims()

	claimsVal, found := claimsMap[keyToCheck]

	if found {
		switch t := claimsVal.(type) {
		case interface{}:
			return filterMatch(filterValues, t)
		case []interface{}:
			return filterMatch(filterValues, t...)
		}
	}

	return false
}

func filterMatch(filterValues map[interface{}]struct{}, paramsToCheck ...interface{}) bool {
	for _, param := range paramsToCheck {
		_, found := filterValues[param]

		if found {
			return true
		}
	}

	return false

}

func (f *FilterGate) ServeHTTP(response http.ResponseWriter, request *http.Request) {

	logger := logging.GetLogger(request.Context())

	method := request.Method
	if method == "GET" {
		filters := f.PrettifyFilters()
		response.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(response, `{"filters": %v}`, filters)
	} else if method == "POST" || method == "PUT" || method == "DELETE" {
		var message filterRequest
		msgBytes, err := ioutil.ReadAll(request.Body)
		request.Body.Close()

		if err != nil {
			logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "could not read request body", logging.ErrorKey(), err)
			xhttp.WriteError(response, http.StatusBadRequest, err)
			return
		}

		err = json.Unmarshal(msgBytes, &message)
		if err != nil {
			logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "could not unmarshal request body", logging.ErrorKey(), err)
			xhttp.WriteError(response, http.StatusBadRequest, err)
			return
		}

		if len(message.Key) == 0 {
			logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "no filter key found")
			xhttp.WriteErrorf(response, http.StatusBadRequest, "missing filter key")
			return
		}

		if len(message.Values) == 0 {
			logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "no filter values found")
			xhttp.WriteErrorf(response, http.StatusBadRequest, "missing filter values")
			return
		}

		if method == "POST" || method == "PUT" {
			if f.allowedFilters != nil {
				_, ok := f.allowedFilters[message.Key]

				if !ok {
					logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "filter key is not allowed", "key: ", message.Key)
					xhttp.WriteErrorf(response, http.StatusBadRequest, "filter key %s is not allowed. Allowed filters: %v", message.Key, f.allowedFilters)
					return
				}
			}

			f.EditFilters(message.Key, message.Values, true)

		} else if method == "DELETE" {
			f.EditFilters(message.Key, message.Values, false)
		}

		logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "gate filters updated", "filters", f.Filters())

		response.WriteHeader(http.StatusOK)
	}
}

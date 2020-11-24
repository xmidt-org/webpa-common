package devicegate

import (
	"sync"

	"github.com/xmidt-org/webpa-common/device"
)

const (
	metadataMapLocation = "metadata_map"
	claimsLocation      = "claims"
)

type DeviceGate interface {
	GetFilters() map[string]interface{}
	GetFilter(key string) (map[interface{}]bool, bool)
	SetFilter(key string, values []interface{}) ([]interface{}, bool)
	DeleteFilter(key string) bool
	AllowedFilters() map[string]bool
	device.Filter
}

type FilterStore map[string]map[interface{}]bool

type FilterGate struct {
	FilterStore    FilterStore
	AllowedFilters map[string]bool

	lock sync.RWMutex
}

type filterRequest struct {
	Key    string
	Values []interface{}
}

func (f *FilterGate) GetFilters() map[string][]interface{} {
	copy := make(map[string][]interface{})

	f.lock.RLock()
	for k, v := range f.FilterStore {
		var filters []interface{}
		for filter := range v {
			filters = append(filters, filter)
		}

		copy[k] = filters
	}
	f.lock.RUnlock()

	return copy
}

func (f *FilterGate) GetFilter(key string) (map[interface{}]bool, bool) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	v, ok := f.FilterStore[key]
	return v, ok

}

func (f *FilterGate) SetFilter(key string, values []interface{}) ([]interface{}, bool) {
	f.lock.Lock()
	defer f.lock.Unlock()

	newValues := make(map[interface{}]bool)

	for _, v := range values {
		newValues[v] = true
	}

	f.FilterStore[key] = newValues

	return values, true
}

func (f *FilterGate) DeleteFilter(key string) bool {
	f.lock.Lock()
	defer f.lock.Unlock()

	_, ok := f.FilterStore[key]

	if ok {
		delete(f.FilterStore, key)
		return true
	}

	return false
}

func (f *FilterGate) AllowConnection(d device.Interface) (bool, device.MatchResult) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	for filterKey, filterValues := range f.FilterStore {

		// check if filter is in claims
		if f.FilterStore.claimsMatch(filterKey, filterValues, d.Metadata()) {
			return false, device.MatchResult{
				Location: metadataMapLocation,
				Key:      filterKey,
			}
		}

		// check if filter is in metadata map
		if f.FilterStore.metadataMapMatch(filterKey, filterValues, d.Metadata()) {
			return false, device.MatchResult{
				Location: claimsLocation,
				Key:      filterKey,
			}
		}

	}

	return true, device.MatchResult{}
}

func (f *FilterStore) metadataMapMatch(keyToCheck string, filterValues map[interface{}]bool, m *device.Metadata) bool {
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

func (f *FilterStore) claimsMatch(keyToCheck string, filterValues map[interface{}]bool, m *device.Metadata) bool {
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

func filterMatch(filterValues map[interface{}]bool, paramsToCheck ...interface{}) bool {
	for _, param := range paramsToCheck {
		_, found := filterValues[param]

		if found {
			return true
		}
	}

	return false

}

func mapToArray(m map[interface{}]bool) []interface{} {

	list := make([]interface{}, len(m))

	i := 0
	for key, _ := range m {
		list[i] = key
		i++
	}

	return list
}

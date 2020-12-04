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
	VisitAll(visit func(string, interface{}))
	GetFilter(key string) (Set, bool)
	SetFilter(key string, values []interface{}) ([]interface{}, bool)
	DeleteFilter(key string) bool
	GetAllowedFilters() (Set, bool)
	device.Filter
}

type Set interface {
	Exists(interface{}) bool
	GetAll() []interface{}
	VisitAll(func(interface{}))
}

type FilterStore map[string]Set

type FilterSet map[interface{}]bool

type FilterGate struct {
	FilterStore    FilterStore
	AllowedFilters Set

	lock sync.RWMutex
}

type filterRequest struct {
	Key    string
	Values []interface{}
}

func (f *FilterGate) VisitAll(visit func(string, interface{})) {
	f.lock.RLock()
	defer f.lock.RUnlock()
	for key, filterValues := range f.FilterStore {
		filterValues.VisitAll(func(v interface{}) {
			visit(key, v)
		})
	}
}

func (f *FilterGate) GetFilter(key string) (Set, bool) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	v, ok := f.FilterStore[key]
	return v, ok

}

// TODO: return set of old values instead?
func (f *FilterGate) SetFilter(key string, values []interface{}) ([]interface{}, bool) {
	f.lock.Lock()
	defer f.lock.Unlock()

	newValues := make(FilterSet)

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

func (f *FilterGate) GetAllowedFilters() (Set, bool) {
	if f.AllowedFilters == nil {
		return f.AllowedFilters, false
	}

	return f.AllowedFilters, true
}

func (f *FilterStore) metadataMapMatch(keyToCheck string, filterValues Set, m *device.Metadata) bool {
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

func (f *FilterStore) claimsMatch(keyToCheck string, filterValues Set, m *device.Metadata) bool {
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

func filterMatch(filterValues Set, paramsToCheck ...interface{}) bool {
	for _, param := range paramsToCheck {
		if filterValues.Exists(param) {
			return true
		}
	}

	return false

}

func (s FilterSet) Exists(key interface{}) bool {
	return s[key]
}

func (s FilterSet) GetAll() []interface{} {
	list := make([]interface{}, len(s))
	for key := range s {
		list = append(list, key)
	}

	return list
}

func (s FilterSet) VisitAll(f func(interface{})) {
	for key := range s {
		f(key)
	}
}

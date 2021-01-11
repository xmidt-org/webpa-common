package devicegate

import (
	"encoding/json"
	"sync"

	"github.com/xmidt-org/webpa-common/device"
)

const (
	metadataMapLocation = "metadata_map"
	claimsLocation      = "claims"
)

// Interface is a gate interface specifically for filtering devices
type Interface interface {
	device.Filter

	// VisitAll applies the given visitor function to each set of filter values
	//
	// No methods on this Interface should be called from within the visitor function, or
	// a deadlock will likely occur.
	VisitAll(visit func(string, Set) bool) int

	// GetFilter returns the set of filter values associated with a filter key and a bool
	// that is true if the key was found, false if it doesn't exist.
	GetFilter(key string) (Set, bool)

	// SetFilter saves the filter values and filter key to filter by. It returns a Set of the old values and a
	// bool that is true if the filter key did not previously exist and false if the filter key had existed beforehand.
	SetFilter(key string, values []interface{}) (Set, bool)

	// DeleteFilter deletes a filter key. This completely removes all filter values associated with that key as well.
	// Returns true if key had existed and values actually deleted, and false if key was not found.
	DeleteFilter(key string) bool

	// GetAllowedFilters returns the set of filters that devices are allowed to be filtered by. Also returns a
	// bool that is true if there are allowed filters set, and false if there aren't (meaning that all filters are allowed)
	GetAllowedFilters() (Set, bool)
}

// Set is an interface that represents a read-only hashset
type Set interface {
	json.Marshaler
	// Has returns true if a value exists in the set, false if it doesn't.
	Has(interface{}) bool

	// VisitAll applies the visitor function to every value in the set.
	VisitAll(func(interface{}))
}

// FilterStore can be used to store filters in the Interface
type FilterStore map[string]Set

// FilterSet is a concrete type that implements the Set interface
type FilterSet struct {
	Set  map[interface{}]bool
	lock sync.RWMutex
}

// FilterGate is a concrete implementation of the Interface
type FilterGate struct {
	FilterStore    FilterStore `json:"filters"`
	AllowedFilters Set         `json:"allowedFilters"`

	lock sync.RWMutex
}

type FilterRequest struct {
	Key    string        `json:"key"`
	Values []interface{} `json:"values"`
}

func (f *FilterGate) VisitAll(visit func(string, Set) bool) int {
	f.lock.RLock()
	defer f.lock.RUnlock()

	visited := 0
	for key, filterValues := range f.FilterStore {
		visited++
		if !visit(key, filterValues) {
			break
		}
	}

	return visited
}

func (f *FilterGate) GetFilter(key string) (Set, bool) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	v, ok := f.FilterStore[key]
	return v, ok

}

func (f *FilterGate) SetFilter(key string, values []interface{}) (Set, bool) {
	f.lock.Lock()
	defer f.lock.Unlock()

	oldValues := f.FilterStore[key]
	newValues := make(map[interface{}]bool)

	for _, v := range values {
		newValues[v] = true
	}

	f.FilterStore[key] = &FilterSet{
		Set: newValues,
	}

	if oldValues == nil {
		return oldValues, true
	}

	return oldValues, false

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
		// check for filter match
		if found, result := f.FilterStore.metadataMatch(filterKey, filterValues, d.Metadata()); found {
			return false, result
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

func (s *FilterSet) Has(key interface{}) bool {
	if s.Set != nil {
		s.lock.RLock()
		defer s.lock.RUnlock()
		return s.Set[key]
	}

	return false
}

func (s *FilterSet) VisitAll(f func(interface{})) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	for key := range s.Set {
		f(key)
	}
}

func (s *FilterSet) MarshalJSON() ([]byte, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	temp := make([]interface{}, 0, len(s.Set))
	for key := range s.Set {
		temp = append(temp, key)
	}

	return json.Marshal(temp)
}

func (f *FilterStore) metadataMatch(keyToCheck string, filterValues Set, m *device.Metadata) (bool, device.MatchResult) {
	var val interface{}
	result := device.MatchResult{
		Key: keyToCheck,
	}
	if metadataVal := m.Load(keyToCheck); metadataVal != nil {
		val = metadataVal
		result.Location = metadataMapLocation
	} else if claimsVal, found := m.Claims()[keyToCheck]; found {
		val = claimsVal
		result.Location = claimsLocation
	}

	if val != nil {
		switch t := val.(type) {
		case []interface{}:
			if filterMatch(filterValues, t...) {
				return true, result
			}
		case interface{}:
			if filterMatch(filterValues, t) {
				return true, result
			}
		}
	}

	return false, device.MatchResult{}
}

// function to check if any params are in a set
func filterMatch(filterValues Set, paramsToCheck ...interface{}) bool {
	for _, param := range paramsToCheck {
		if filterValues.Has(param) {
			return true
		}
	}

	return false

}

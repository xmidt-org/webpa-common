package devicegate

import (
	"fmt"
	"strings"
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

	// Has returns true if a value exists in the set, false if it doesn't.
	Has(interface{}) bool

	// VisitAll applies the visitor function to every value in the set.
	VisitAll(func(interface{}))

	// String returns a string representation of the set.
	String() string
}

// FilterStore can be used to store filters in the Interface
type FilterStore map[string]Set

// FilterSet is a concrete type that implements the Set interface
type FilterSet map[interface{}]bool

// FilterGate is a concrete implementation of the Interface
type FilterGate struct {
	FilterStore    FilterStore
	AllowedFilters Set

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
	newValues := make(FilterSet)

	for _, v := range values {
		newValues[v] = true
	}

	f.FilterStore[key] = newValues

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

func (s FilterSet) Has(key interface{}) bool {
	return s[key]
}

func (s FilterSet) VisitAll(f func(interface{})) {
	for key := range s {
		f(key)
	}
}

func (s FilterSet) String() string {
	var b strings.Builder
	b.WriteString("[")

	var needsComma bool
	s.VisitAll(func(v interface{}) {
		if needsComma {
			b.WriteString(", ")
			needsComma = false
		}

		fmt.Fprintf(&b, `"%v"`, v)
		needsComma = true
	})

	b.WriteString("]")
	return b.String()
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

// function to check if any params are in a set
func filterMatch(filterValues Set, paramsToCheck ...interface{}) bool {
	for _, param := range paramsToCheck {
		if filterValues.Has(param) {
			return true
		}
	}

	return false

}
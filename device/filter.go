package device

import "sync"

const (
	metadataMapLocation = "metadata_map"
	claimsLocation      = "claims"
)

var (
	emptyVal  = struct{}{}
	FilterMap = map[string]struct{}{
		PartnerIDClaimKey: emptyVal,
	}
)

type Filter interface {
	AllowConnection(d *device) (bool, matchResult)
	GetFilters() map[string]map[string]struct{}
	EditFilters(key string, values []string, add bool) bool
}

type filtersStore struct {
	filters map[string]map[string]struct{}
	lock    sync.RWMutex
}

type matchResult struct {
	location string
	key      string
}

func (f *filtersStore) Prettify() map[string][]string {
	copy := make(map[string][]string)

	f.lock.RLock()
	for k, v := range f.filters {
		var filters []string
		for filter := range v {
			filters = append(filters, filter)
		}

		copy[k] = filters
	}
	f.lock.RUnlock()

	return copy
}

func (f *filtersStore) GetFilters() map[string]map[string]struct{} {
	filtersCopy := make(map[string]map[string]struct{})

	f.lock.RLock()
	for key, val := range f.filters {
		valCopy := make(map[string]struct{})

		for k, v := range val {
			valCopy[k] = v
		}

		filtersCopy[key] = valCopy
	}
	f.lock.RUnlock()

	return filtersCopy
}

//allows for adding and removing filters
//Will completely overwrite current filters in place
func (f *filtersStore) EditFilters(key string, values []string, add bool) bool {
	f.lock.Lock()
	defer f.lock.Unlock()
	if add {
		valuesMap := make(map[string]struct{})

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

func (f *filtersStore) AllowConnection(d *device) (bool, matchResult) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	for filterKey, filterValues := range f.filters {

		// check if filter is in claims
		if claimsMatch(filterKey, filterValues, d.Metadata()) {
			return false, matchResult{
				location: metadataMapLocation,
				key:      filterKey,
			}
		}

		// check if filter is in metadata map
		if metadataMapMatch(filterKey, filterValues, d.Metadata()) {
			return false, matchResult{
				location: claimsLocation,
				key:      filterKey,
			}
		}

	}

	return true, matchResult{}
}

func metadataMapMatch(keyToCheck string, filterValues map[string]struct{}, m *Metadata) bool {
	metadataVal := m.Load(keyToCheck)
	if metadataVal != nil {
		switch t := metadataVal.(type) {
		case string:
			return filterMatch(filterValues, t)
		case []string:
			return filterMatch(filterValues, t...)

		}
	}

	return false

}

func claimsMatch(keyToCheck string, filterValues map[string]struct{}, m *Metadata) bool {
	claimsMap := m.Claims()

	claimsVal, found := claimsMap[keyToCheck]

	if found {
		switch t := claimsVal.(type) {
		case string:
			return filterMatch(filterValues, t)
		case []string:
			return filterMatch(filterValues, t...)
		}
	}

	return false
}

func filterMatch(filterValues map[string]struct{}, paramsToCheck ...string) bool {
	for _, param := range paramsToCheck {
		_, found := filterValues[param]

		if found {
			return true
		}
	}

	return false

}

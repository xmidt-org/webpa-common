package monitor

import (
	"sort"

	"github.com/xmidt-org/webpa-common/service"
)

// Filter represents a preprocessing strategy for discovered service instances
type Filter func([]string) []string

// NopFilter does nothing.  It returns the slice of instances as is.
func NopFilter(i []string) []string {
	return i
}

// NewNormalizeFilter returns a Filter that uses service.NormalizeInstance to ensure that each instance
// is a valid URI with scheme and port (where applicable).  The defaultScheme is used if an instance has
// no scheme, e.g. "localhost:8080".
func NewNormalizeFilter(defaultScheme string) Filter {
	return func(original []string) []string {
		if len(original) == 0 {
			return original
		}

		filtered := make([]string, 0, len(original))
		for _, o := range original {
			if normalized, err := service.NormalizeInstance(defaultScheme, o); err == nil {
				filtered = append(filtered, normalized)
			}
		}

		sort.Strings(filtered)
		return filtered
	}
}

var defaultFilter = NewNormalizeFilter(service.DefaultScheme)

// DefaultFilter returns the global default Filter instance.
func DefaultFilter() Filter {
	return defaultFilter
}

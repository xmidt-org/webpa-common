package monitor

import (
	"sort"

	"github.com/Comcast/webpa-common/service"
)

// Filter represents a preprocessing strategy for discovered service instances
type Filter func([]string) []string

// NopFilter does nothing.  It returns the slice of instances as is.
func NopFilter(i []string) []string {
	return i
}

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

func DefaultFilter() Filter {
	return defaultFilter
}

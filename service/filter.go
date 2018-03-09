package service

import (
	"sort"
	"strings"
)

// Filter represents a preprocessing strategy for discovered service instances
type Filter func([]string) []string

// NopFilter does nothing.  It returns the slice of instances as is.
func NopFilter(i []string) []string {
	return i
}

// DefaultFilter removes any blank strings (not just empty strings) from the array, which
// seems to happen with some service discovery backends (e.g. zookeeper).  The returned slice
// is distinct from the original and will be sorted consistently.
func DefaultFilter(original []string) []string {
	filtered := make([]string, 0, len(original))

	for _, o := range original {
		f := strings.TrimSpace(o)
		if len(f) > 0 {
			filtered = append(filtered, f)
		}
	}

	sort.Strings(filtered)
	return filtered
}

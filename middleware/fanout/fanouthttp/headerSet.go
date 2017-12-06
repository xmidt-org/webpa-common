package fanouthttp

import (
	"net/http"
	"net/textproto"
	"strings"
)

// HeaderSet represents a collection of header names which provides a number of useful operations on
// http.Header objects.  This type implements sort.Interface and heap.Interface.
type HeaderSet []string

// NewHeaderSet returns a set of headers initialized with the given names, via Add.
func NewHeaderSet(names ...string) HeaderSet {
	var hs HeaderSet
	hs.Add(names...)
	return hs
}

func (hs HeaderSet) String() string {
	return strings.Join(hs, ",")
}

// Add inserts zero or more headers into this set.  Each header name is first canonicalized
// in the same way as with http.Header.  This method makes no attempt at deduplication.
func (hs *HeaderSet) Add(names ...string) {
	for _, n := range names {
		n = textproto.CanonicalMIMEHeaderKey(n)
		*hs = append(*hs, n)
	}
}

// Filter takes each header in this set from the source and sets it on the target.
// If this set is empty, then this method does nothing.  If target is nil, a new
// http.Header is created to hold the filtered headers.  The target http.Header
// is returned, even if no filtering took place.
func (hs HeaderSet) Filter(target, source http.Header) http.Header {
	if len(hs) > 0 {
		if target == nil {
			target = make(http.Header, len(hs))
		}

		for _, h := range hs {
			if s, ok := source[h]; ok {
				target[h] = s
			}
		}
	}

	return target
}

package fanout

import "net/http"

// ShouldTerminateFunc is a predicate for determining if a fanout should terminate early given the results of
// a single HTTP transaction.
type ShouldTerminateFunc func(*http.Response, error) bool

// DefaultShouldTerminate is the default strategy for determining if an HTTP transaction should result
// in early termination of the fanout.  This function returns true if and only if fanoutResponse is non-nil
// and has a status code less than 400.
func DefaultShouldTerminate(fanoutResponse *http.Response, _ error) bool {
	return fanoutResponse != nil && fanoutResponse.StatusCode < 400
}

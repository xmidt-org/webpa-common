package xhttptest

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"

	"github.com/stretchr/testify/mock"
)

// ExpectedResponse is a tuple of the expected return values from transactor.Do.  This struct provides
// a simple unit to build table-driven tests from.
type ExpectedResponse struct {
	Response *http.Response
	Err      error
}

// TransactCall is a stretchr mock Call with some extra behavior to make mocking out HTTP client behavior easier
type TransactCall struct {
	*mock.Call
}

func (dc *TransactCall) Respond(er ExpectedResponse) *TransactCall {
	dc.Return(er.Response, er.Err)
	return dc
}

// RespondWithError is a convenience for setting a return of (nil, err).
// Note that if err is nil, this may result in strange behavior as returning (nil, nil)
// violates the general constract of http.Client.Do and http.RoundTripper.RoundTrip.
func (dc *TransactCall) RespondWithError(err error) *TransactCall {
	dc.Return((*http.Response)(nil), err)
	return dc
}

// RespondWith is a convenience for setting a return of (response, nil).  Note that if response is
// nil, this may result in strange behavior as returning (nil, nil) violates the general
// constract of http.Client.Do and http.RoundTripper.RoundTrip.
func (dc *TransactCall) RespondWith(response *http.Response) *TransactCall {
	dc.Return(response, nil)
	return dc
}

// MockTransactor is a stretchr mock for the Do method of an HTTP client or round tripper.
// This mock extends the behavior of a stretchr mock in a few ways that make clientside
// HTTP behavior easier to mock.
//
// This type implements the http.RoundTripper interface, and provides a Do method that can
// implement a subset interface of http.Client.
type MockTransactor struct {
	mock.Mock
}

// Do is a mocked HTTP transaction call.  Use On or OnRequest to setup behaviors for this method.
func (mt *MockTransactor) Do(request *http.Request) (*http.Response, error) {
	arguments := mt.Called(request)
	response, _ := arguments.Get(0).(*http.Response)
	return response, arguments.Error(1)
}

// RoundTrip is a mocked HTTP transaction call.  Use On or OnRoundTrip to setup behaviors for this method.
func (mt *MockTransactor) RoundTrip(request *http.Request) (*http.Response, error) {
	arguments := mt.Called(request)
	response, _ := arguments.Get(0).(*http.Response)
	return response, arguments.Error(1)
}

// OnDo sets an On("Do", ...) with the given matchers for a request.  The returned Call has some
// augmented behavior for setting responses.
func (mt *MockTransactor) OnDo(matchers ...func(*http.Request) bool) *TransactCall {
	call := mt.On("Do", mock.MatchedBy(func(candidate *http.Request) bool {
		for _, matcher := range matchers {
			if !matcher(candidate) {
				return false
			}
		}

		return true
	}))

	return &TransactCall{call}
}

// OnRoundTrip sets an On("Do", ...) with the given matchers for a request.  The returned Call has some
// augmented behavior for setting responses.
func (mt *MockTransactor) OnRoundTrip(matchers ...func(*http.Request) bool) *TransactCall {
	call := mt.On("RoundTrip", mock.MatchedBy(func(candidate *http.Request) bool {
		for _, matcher := range matchers {
			if !matcher(candidate) {
				return false
			}
		}

		return true
	}))

	return &TransactCall{call}
}

// MatchMethod returns a request matcher that verifies each request has a specific method
func MatchMethod(expected string) func(*http.Request) bool {
	return func(r *http.Request) bool {
		return strings.EqualFold(expected, r.Method)
	}
}

// MatchURL returns a request matcher that verifies each request has an exact URL.
func MatchURL(expected *url.URL) func(*http.Request) bool {
	return func(r *http.Request) bool {
		if expected == r.URL {
			return true
		}

		if expected == nil || r.URL == nil {
			return false
		}

		return *expected == *r.URL
	}
}

// MatchURLString returns a request matcher that verifies the request's URL translates to the given string.
func MatchURLString(expected string) func(*http.Request) bool {
	return func(r *http.Request) bool {
		if r.URL == nil {
			return len(expected) == 0
		}

		return expected == r.URL.String()
	}
}

// MatchBody returns a request matcher that verifies each request has an exact body.
// The body is consumed, but then replaced so that downstream code can still access the body.
func MatchBody(expected []byte) func(*http.Request) bool {
	return func(r *http.Request) bool {
		if r.Body == nil {
			return len(expected) == 0
		}

		actual, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(fmt.Errorf("Error while read request body for matching: %s", err))
		}

		// replace the body so other test code can reread it
		r.Body = ioutil.NopCloser(bytes.NewReader(actual))

		if len(actual) != len(expected) {
			return false
		}

		for i := 0; i < len(actual); i++ {
			if actual[i] != expected[i] {
				return false
			}
		}

		return true
	}
}

// MatchHeader returns a request matcher that matches against a request header
func MatchHeader(name, expected string) func(*http.Request) bool {
	return func(r *http.Request) bool {
		// allow for requests created by test code that instantiates the request directly
		if r.Header == nil {
			return false
		}

		values := r.Header[textproto.CanonicalMIMEHeaderKey(name)]
		if len(values) == 0 {
			return len(expected) == 0
		}

		for _, actual := range values {
			if actual == expected {
				return true
			}
		}

		return false
	}
}

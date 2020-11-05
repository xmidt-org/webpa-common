/**
 * Copyright 2020 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package basculechecks

import (
	"errors"

	"github.com/goph/emperror"
	"github.com/xmidt-org/bascule"
)

var (
	ErrNilDefaultChecker = errors.New("default checker cannot be nil")
	ErrEmptyEndpoint     = errors.New("endpoint provided is empty")
)

// CapabilitiesMap runs a capability check based on the value of the parsedURL,
// which is the key to the CapabilitiesMap's map.  The parsedURL is expected to
// be some regex values, allowing for bucketing of urls that contain some kind
// of ID or otherwise variable portion of a URL.
type CapabilitiesMap struct {
	checkers       map[string]CapabilityChecker
	defaultChecker CapabilityChecker
}

// NewCapabilitiesMap validates the default Checker and sets up the map to be
// non-nil in order to create a good CapabilitiesMap struct.
func NewCapabilitiesMap(checkers map[string]CapabilityChecker, defaultChecker CapabilityChecker) (*CapabilitiesMap, error) {
	if defaultChecker == nil {
		return nil, ErrNilDefaultChecker
	}
	if checkers == nil {
		checkers = make(map[string]CapabilityChecker)
	}
	return &CapabilitiesMap{
		checkers:       checkers,
		defaultChecker: defaultChecker,
	}, nil
}

// Check uses the parsed endpoint value to determine which CapabilityChecker to
// run against the capabilities in the auth provided.  If there is no
// CapabilityChecker for the endpoint, the default is used.  As long as one
// capability is found to be authorized by the CapabilityChecker, no error is
// returned.
func (c CapabilitiesMap) Check(auth bascule.Authentication, vs ParsedValues) (string, error) {
	if auth.Token == nil {
		return TokenMissingValues, ErrNoToken
	}

	if auth.Request.URL == nil {
		return TokenMissingValues, ErrNoURL
	}

	if vs.Endpoint == "" {
		return EmptyParsedURL, ErrEmptyEndpoint
	}

	capabilities, reason, err := getCapabilities(auth.Token.Attributes())
	if err != nil {
		return reason, err
	}

	// determine which CapabilityChecker to use.
	checker, ok := c.checkers[vs.Endpoint]
	if !ok || checker == nil {
		checker = c.defaultChecker
	}
	reqURL := auth.Request.URL.EscapedPath()
	method := auth.Request.Method

	// if one of the capabilities is good, then the request is authorized
	// for this endpoint.
	for _, capability := range capabilities {
		if checker.Authorized(capability, reqURL, method) {
			return "", nil
		}
	}

	return NoCapabilitiesMatch, emperror.With(ErrNoValidCapabilityFound,
		"capabilitiesFound", capabilities, "request URL", reqURL,
		"request method", method, "parsed URL", vs.Endpoint)

}

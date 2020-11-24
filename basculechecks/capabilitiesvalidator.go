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
	"context"
	"errors"
	"fmt"

	"github.com/goph/emperror"
	"github.com/spf13/cast"
	"github.com/xmidt-org/bascule"
)

var (
	ErrNoVals                 = errors.New("expected at least one value")
	ErrNoAuth                 = errors.New("couldn't get request info: authorization not found")
	ErrNoToken                = errors.New("no token found in Auth")
	ErrNoValidCapabilityFound = errors.New("no valid capability for endpoint")
	ErrNilAttributes          = errors.New("nil attributes interface")
	ErrNoURL                  = errors.New("invalid URL found in Auth")
)

const (
	CapabilityKey = "capabilities"
)

var (
	partnerKeys = []string{"allowedResources", "allowedPartners"}
)

func PartnerKeys() []string {
	return partnerKeys
}

// CapabilityChecker is an object that can determine if a capability provides
// authorization to the endpoint.
type CapabilityChecker interface {
	Authorized(string, string, string) bool
}

// CapabilitiesValidator checks the capabilities provided in a
// bascule.Authentication object to determine if a request is authorized.  It
// can also provide a function to be used in authorization middleware that
// pulls the Authentication object from a context before checking it.
type CapabilitiesValidator struct {
	Checker CapabilityChecker
}

// CreateValidator creates a function that determines whether or not a
// client is authorized to make a request to an endpoint.  It uses the
// bascule.Authentication from the context to get the information needed by the
// CapabilityChecker to determine authorization.
func (c CapabilitiesValidator) CreateValidator(errorOut bool) bascule.ValidatorFunc {
	return func(ctx context.Context, _ bascule.Token) error {
		auth, ok := bascule.FromContext(ctx)
		if !ok {
			if errorOut {
				return ErrNoAuth
			}
			return nil
		}

		_, err := c.Check(auth, ParsedValues{})
		if err != nil && errorOut {
			return err
		}

		return nil
	}
}

// Check takes the needed values out of the given Authentication object in
// order to determine if a request is authorized.  It determines this through
// iterating through each capability and calling the CapabilityChecker.  If no
// capability authorizes the client for the given endpoint and method, it is
// unauthorized.
func (c CapabilitiesValidator) Check(auth bascule.Authentication, _ ParsedValues) (string, error) {
	if auth.Token == nil {
		return TokenMissingValues, ErrNoToken
	}
	vals, reason, err := getCapabilities(auth.Token.Attributes())
	if err != nil {
		return reason, err
	}

	if auth.Request.URL == nil {
		return TokenMissingValues, ErrNoURL
	}
	reqURL := auth.Request.URL.EscapedPath()
	method := auth.Request.Method
	err = c.checkCapabilities(vals, reqURL, method)
	if err != nil {
		return NoCapabilitiesMatch, err
	}
	return "", nil
}

// checkCapabilities uses a CapabilityChecker to check if each capability
// provided is authorized.  If an authorized capability is found, no error is
// returned.
func (c CapabilitiesValidator) checkCapabilities(capabilities []string, reqURL string, method string) error {
	for _, val := range capabilities {
		if c.Checker.Authorized(val, reqURL, method) {
			return nil
		}
	}
	return emperror.With(ErrNoValidCapabilityFound, "capabilitiesFound", capabilities, "urlToMatch", reqURL, "methodToMatch", method)

}

// getCapabilities runs some error checks while getting the list of
// capabilities from the attributes.
func getCapabilities(attributes bascule.Attributes) ([]string, string, error) {
	if attributes == nil {
		return []string{}, UndeterminedCapabilities, ErrNilAttributes
	}

	val, ok := attributes.Get(CapabilityKey)
	if !ok {
		return []string{}, UndeterminedCapabilities, fmt.Errorf("couldn't get capabilities using key %v", CapabilityKey)
	}

	vals, err := cast.ToStringSliceE(val)
	if err != nil {
		return []string{}, UndeterminedCapabilities, fmt.Errorf("capabilities \"%v\" not the expected string slice: %v", val, err)
	}

	if len(vals) == 0 {
		return []string{}, EmptyCapabilitiesList, ErrNoVals
	}

	return vals, "", nil

}

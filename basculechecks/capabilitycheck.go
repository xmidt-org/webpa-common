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
	"fmt"
	"regexp"
	"strings"
)

// ConstCheck is a basic capability checker that determines a capability is
// authorized if it matches the ConstCheck's string.
type ConstCheck string

// Authorized validates the capability provided against the stored string.
func (c ConstCheck) Authorized(capability, _, _ string) bool {
	return string(c) == capability
}

// EndpointRegexCheck uses a regular expression to validate an endpoint and
// method provided in a capability against the endpoint hit and method used for
// the request.
type EndpointRegexCheck struct {
	prefixToMatch   *regexp.Regexp
	acceptAllMethod string
}

// NewEndpointRegexCheck creates an object that implements the
// CapabilityChecker interface.  It takes a prefix that is expected at the
// beginning of a capability and a string that, if provided in the capability,
// authorizes all methods for that endpoint.  After the prefix, the
// EndpointRegexCheck expects there to be an endpoint regular expression and an
//http method - separated by a colon. The expected format of a capability is:
// <prefix><endpoint regex>:<method>
func NewEndpointRegexCheck(prefix string, acceptAllMethod string) (EndpointRegexCheck, error) {
	matchPrefix, err := regexp.Compile("^" + prefix + "(.+):(.+?)$")
	if err != nil {
		return EndpointRegexCheck{}, fmt.Errorf("failed to compile prefix [%v]: %w", prefix, err)
	}

	e := EndpointRegexCheck{
		prefixToMatch:   matchPrefix,
		acceptAllMethod: acceptAllMethod,
	}
	return e, nil
}

// Authorized checks the capability against the endpoint hit and method used.
// If the capability has the correct prefix and is meant to be used with the
// method provided to access the endpoint provided, it is authorized.
func (e EndpointRegexCheck) Authorized(capability string, urlToMatch string, methodToMatch string) bool {
	matches := e.prefixToMatch.FindStringSubmatch(capability)

	if matches == nil || len(matches) < 2 {
		return false
	}

	method := matches[2]
	if method != e.acceptAllMethod && method != strings.ToLower(methodToMatch) {
		return false
	}

	re, err := regexp.Compile(matches[1]) //url regex that capability grants access to
	if err != nil {
		return false
	}

	matchIdxs := re.FindStringIndex(urlToMatch)
	if matchIdxs == nil || matchIdxs[0] != 0 {
		return false
	}

	return true
}

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

type ConstCheck string

func (c ConstCheck) Authorized(capability, _, _ string) bool {
	return string(c) == capability
}

type EndpointRegexCheck struct {
	prefixToMatch   *regexp.Regexp
	acceptAllMethod string
}

// NewCapabilityChecker creates an object that produces a check on capabilities
// in bascule tokens, to be run by the bascule enforcer middleware.
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

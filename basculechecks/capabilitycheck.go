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
	"regexp"
	"strings"
)

type ConstCheck string

func (c ConstCheck) Authorized(capability, _, _ string) bool {
	return string(c) == capability
}

type EndpointRegexCheck struct {
	PrefixToMatch   *regexp.Regexp
	AcceptAllMethod string
}

func (e EndpointRegexCheck) Authorized(capability string, urlToMatch string, methodToMatch string) bool {
	matches := e.PrefixToMatch.FindStringSubmatch(capability)

	if matches == nil || len(matches) < 2 {
		return false
	}

	method := matches[2]
	if method != e.AcceptAllMethod && method != strings.ToLower(methodToMatch) {
		return false
	}

	re := regexp.MustCompile(matches[1]) //url regex that capability grants access to
	matchIdxs := re.FindStringIndex(urlToMatch)
	if matchIdxs == nil || matchIdxs[0] != 0 {
		return false
	}

	return true
}

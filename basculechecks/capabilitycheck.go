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
	"regexp"
	"strings"

	"github.com/goph/emperror"
)

var (
	ErrNilPrefix = errors.New("prefix regular expression cannot be nil")
)

type constCheck struct {
	val string
}

func (c constCheck) Authorized(capability string, _ string, _ string) bool {
	return c.val == capability
}

func NewConstCheck(v string) constCheck {
	return constCheck{
		val: v,
	}
}

type endpointRegexCheck struct {
	prefixToMatch   *regexp.Regexp
	acceptAllMethod string
}

func (e *endpointRegexCheck) Authorized(capability string, urlToMatch string, methodToMatch string) bool {
	matches := e.prefixToMatch.FindStringSubmatch(capability)

	if matches == nil || len(matches) < 2 {
		return false
	}

	method := matches[2]
	if method != e.acceptAllMethod && method != strings.ToLower(methodToMatch) {
		return false
	}

	re := regexp.MustCompile(matches[1]) //url regex that capability grants access to
	matchIdxs := re.FindStringIndex(urlToMatch)
	if matchIdxs == nil || matchIdxs[0] != 0 {
		return false
	}

	return true
}

// NewEndpointRegexCheck creates an object that produces a check on capabilities
// in bascule tokens, to be run by the bascule enforcer middleware.
func NewEndpointRegexCheck(prefix *regexp.Regexp, acceptAllMethod string) (*endpointRegexCheck, error) {
	if prefix == nil {
		return nil, ErrNilPrefix
	}

	e := endpointRegexCheck{
		prefixToMatch:   prefix,
		acceptAllMethod: acceptAllMethod,
	}
	return &e, nil
}

// NewEndpointRegexCheckFromString creates an object that produces a check on capabilities
// in bascule tokens, to be run by the bascule enforcer middleware.
func NewEndpointRegexCheckFromString(prefix string, acceptAllMethod string) (*endpointRegexCheck, error) {
	matchPrefix, err := regexp.Compile("^" + prefix + "(.+):(.+?)$")
	if err != nil {
		return nil, emperror.WrapWith(err, "failed to compile prefix given", "prefix", prefix)
	}

	e := endpointRegexCheck{
		prefixToMatch:   matchPrefix,
		acceptAllMethod: acceptAllMethod,
	}
	return &e, nil
}

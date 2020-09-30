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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConstCapabilityChecker(t *testing.T) {
	var v interface{}
	v = ConstCheck("test")
	_, ok := v.(CapabilityChecker)
	assert.True(t, ok)
}

func TestConstCheck(t *testing.T) {
	tests := []struct {
		description string
		capability  string
		okExpected  bool
	}{
		{
			description: "Success",
			capability:  "perfectmatch",
			okExpected:  true,
		},
		{
			description: "Not a Match",
			capability:  "meh",
			okExpected:  false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			c := ConstCheck("perfectmatch")
			ok := c.Authorized(tc.capability, "ignored1", "ignored2")
			assert.Equal(tc.okExpected, ok)
		})
	}
}

func TestEndpointRegexCapabilityChecker(t *testing.T) {
	assert := assert.New(t)
	var v interface{}
	v, err := NewEndpointRegexCheck("test", "")
	assert.Nil(err)
	_, ok := v.(CapabilityChecker)
	assert.True(ok)
}
func TestNewEndpointRegexError(t *testing.T) {
	e, err := NewEndpointRegexCheck(`\M`, "")
	assert := assert.New(t)
	assert.Empty(e)
	assert.NotNil(err)
}

func TestEndpointRegexCheck(t *testing.T) {
	tests := []struct {
		description     string
		prefix          string
		acceptAllMethod string
		capability      string
		url             string
		method          string
		okExpected      bool
	}{
		{
			description:     "Success",
			prefix:          "a:b:c:",
			acceptAllMethod: "all",
			capability:      "a:b:c:.*:get",
			url:             "/test/ffff//",
			method:          "get",
			okExpected:      true,
		},
		{
			description: "No Match Error",
			prefix:      "a:b:c:",
			capability:  "a:.*:get",
			method:      "get",
		},
		{
			description:     "Wrong Method Error",
			prefix:          "a:b:c:",
			acceptAllMethod: "all",
			capability:      "a:b:c:.*:get",
			method:          "post",
		},
		{
			description:     "Regex Doesn't Compile Error",
			prefix:          "a:b:c:",
			acceptAllMethod: "all",
			capability:      `a:b:c:\M:get`,
			method:          "get",
		},
		{
			description:     "URL Doesn't Match Capability Error",
			prefix:          "a:b:c:",
			acceptAllMethod: "all",
			capability:      "a:b:c:[A..Z]+:get",
			url:             "1111",
			method:          "get",
		},
		{
			description:     "URL Capability Match Wrong Location Error",
			prefix:          "a:b:c:",
			acceptAllMethod: "all",
			capability:      "a:b:c:[A..Z]+:get",
			url:             "11AAAAA",
			method:          "get",
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			e, err := NewEndpointRegexCheck(tc.prefix, tc.acceptAllMethod)
			require.Nil(err)
			require.NotEmpty(e)
			ok := e.Authorized(tc.capability, tc.url, tc.method)
			assert.Equal(tc.okExpected, ok)
		})
	}
}

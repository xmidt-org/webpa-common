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
)

func TestNewCapabilitiesMap(t *testing.T) {
	goodCheckers := map[string]CapabilityChecker{
		"a":      ConstCheck("meh"),
		"bcedef": ConstCheck("yay"),
		"all":    ConstCheck("good"),
	}
	emptyCheckers := map[string]CapabilityChecker{}
	goodDefault := ConstCheck("default checker")
	tests := []struct {
		description    string
		goodDefault    bool
		checkersMap    map[string]CapabilityChecker
		expectedStruct *CapabilitiesMap
		expectedErr    error
	}{
		{
			description: "Success",
			goodDefault: true,
			checkersMap: goodCheckers,
			expectedStruct: &CapabilitiesMap{
				checkers:       goodCheckers,
				defaultChecker: goodDefault,
			},
		},
		{
			description: "Success with Empty Checkers",
			goodDefault: true,
			checkersMap: emptyCheckers,
			expectedStruct: &CapabilitiesMap{
				checkers:       emptyCheckers,
				defaultChecker: goodDefault,
			},
		},
		{
			description: "Success with Nil Checkers",
			goodDefault: true,
			checkersMap: nil,
			expectedStruct: &CapabilitiesMap{
				checkers:       emptyCheckers,
				defaultChecker: goodDefault,
			},
		},
		{
			description:    "Nil Default Error",
			checkersMap:    goodCheckers,
			expectedStruct: nil,
			expectedErr:    ErrNilDefaultChecker,
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			var d CapabilityChecker
			if tc.goodDefault {
				d = goodDefault
			}
			c, err := NewCapabilitiesMap(tc.checkersMap, d)
			assert.Equal(tc.expectedStruct, c)
			assert.Equal(tc.expectedErr, err)
		})
	}
}

func TestCapabilitiesMapCheck(t *testing.T) {
	// TODO: fill in this test.
}

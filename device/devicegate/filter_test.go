// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package devicegate

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/webpa-common/v2/device"
)

func TestFilterGateAllowConnection(t *testing.T) {
	assert := assert.New(t)

	metadata := new(device.Metadata)
	metadata.SetClaims(map[string]interface{}{
		"partner-id": "random-partner",
	})
	metadata.Store("random-key", "abc")

	tests := []struct {
		description string
		filters     map[string]map[interface{}]bool
		canPass     bool
	}{
		{
			description: "Allow",
			canPass:     true,
			filters: map[string]map[interface{}]bool{
				"partner-id": map[interface{}]bool{
					"comcast": true,
				},
			},
		},
		{
			description: "Deny-Filter Match in Claims",
			canPass:     false,
			filters: map[string]map[interface{}]bool{
				"partner-id": map[interface{}]bool{
					"comcast":        true,
					"random-partner": true,
				},
			},
		},
		{
			description: "Deny-Filter Match in Metadata Store",
			canPass:     false,
			filters: map[string]map[interface{}]bool{
				"random-key": map[interface{}]bool{
					"abc":    true,
					"random": true,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			mockDevice := new(device.MockDevice)

			// nolint: typecheck
			mockDevice.On("Metadata").Return(metadata)

			filterStore := make(FilterStore)

			for key, values := range tc.filters {
				fs := FilterSet{
					Set: values,
				}

				filterStore[key] = &fs
			}

			fg := FilterGate{
				FilterStore: filterStore,
			}

			canPass, matchResult := fg.AllowConnection(mockDevice)
			assert.Equal(tc.canPass, canPass)

			if !tc.canPass {
				assert.NotEmpty(matchResult.Location)
				assert.NotEmpty(matchResult.Key)
			}

		})
	}
}

func TestGetSetFilter(t *testing.T) {
	assert := assert.New(t)
	fg := FilterGate{
		FilterStore: make(FilterStore),
	}

	tests := []struct {
		description   string
		keyToSet      string
		valuesToSet   []interface{}
		keyToGet      string
		expectedSet   Set
		expectedFound bool
	}{
		{
			description:   "Add",
			keyToSet:      "test",
			valuesToSet:   []interface{}{"test", "test1"},
			keyToGet:      "test",
			expectedSet:   &FilterSet{Set: map[interface{}]bool{"test": true, "test1": true}},
			expectedFound: true,
		},
		{
			description:   "Update",
			keyToSet:      "test",
			valuesToSet:   []interface{}{"random-value"},
			keyToGet:      "test",
			expectedSet:   &FilterSet{Set: map[interface{}]bool{"random-value": true}},
			expectedFound: true,
		},
		{
			description: "Not Found",
			keyToGet:    "key-no-exist",
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			if len(tc.keyToSet) > 0 {
				fg.SetFilter(tc.keyToSet, tc.valuesToSet)
			}

			getResult, found := fg.GetFilter(tc.keyToGet)

			assert.Equal(tc.expectedFound, found)
			assert.Equal(tc.expectedSet, getResult)
		})
	}
}

func TestDeleteFilter(t *testing.T) {
	assert := assert.New(t)

	fg := FilterGate{
		FilterStore: make(FilterStore),
	}

	tests := []struct {
		description  string
		keyToDelete  string
		expectedBool bool
	}{
		{
			description:  "Delete existing key",
			keyToDelete:  "test",
			expectedBool: true,
		},
		{
			description:  "Delete non-existent key",
			keyToDelete:  "random-key",
			expectedBool: false,
		},
	}

	fg.SetFilter("test", []interface{}{"test1", "test2"})
	fg.SetFilter("key", []interface{}{123, 456})

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			deleted := fg.DeleteFilter(tc.keyToDelete)

			assert.Equal(tc.expectedBool, deleted)
			assert.Nil(fg.GetFilter(tc.keyToDelete))
		})
	}
}

func TestGetAllowedFilters(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		description    string
		allowedFilters *FilterSet
		setExists      bool
	}{
		{
			description: "Non-empty allowed filters set",
			allowedFilters: &FilterSet{Set: map[interface{}]bool{
				"test":          true,
				"random-filter": true,
			}},
			setExists: true,
		},
		{
			description:    "Empty allowed filters set",
			allowedFilters: &FilterSet{Set: map[interface{}]bool{}},
			setExists:      true,
		},
		{
			description:    "Nil allowed filters set",
			allowedFilters: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			var fg FilterGate
			if tc.allowedFilters != nil {
				fg = FilterGate{
					AllowedFilters: tc.allowedFilters,
				}
			}

			filters, isSet := fg.GetAllowedFilters()

			assert.Equal(tc.setExists, isSet)

			if tc.setExists {
				assert.NotNil(filters)
			} else {
				assert.Nil(filters)
			}
		})
	}
}

func TestMetadataMatch(t *testing.T) {
	assert := assert.New(t)
	tests := []struct {
		description         string
		claims              map[string]interface{}
		store               map[string]interface{}
		filterKey           string
		filterValues        Set
		expectedMatch       bool
		expectedMatchResult device.MatchResult
	}{
		{
			description: "claims match",
			claims: map[string]interface{}{
				"test":  "test1",
				"test2": "random-value",
			},
			filterKey: "test",
			filterValues: &FilterSet{Set: map[interface{}]bool{
				"test1": true,
				"test2": true,
			}},
			expectedMatch:       true,
			expectedMatchResult: device.MatchResult{Location: claimsLocation, Key: "test"},
		},
		{
			description: "store match",
			store: map[string]interface{}{
				"test":  "test1",
				"test2": "random-value",
			},
			filterKey: "test",
			filterValues: &FilterSet{Set: map[interface{}]bool{
				"test1": true,
				"test2": true,
			}},
			expectedMatch:       true,
			expectedMatchResult: device.MatchResult{Location: metadataMapLocation, Key: "test"},
		},
		{
			description: "array match",
			claims: map[string]interface{}{
				"test":  []interface{}{"test1", "random"},
				"test2": "random-value",
			},
			filterKey: "test",
			filterValues: &FilterSet{Set: map[interface{}]bool{
				"test1": true,
				"test2": true,
			}},
			expectedMatch:       true,
			expectedMatchResult: device.MatchResult{Location: claimsLocation, Key: "test"},
		},
		{
			description: "no value match",
			claims: map[string]interface{}{
				"test":  []interface{}{"test1", "random"},
				"test2": "random-value",
			},
			store: map[string]interface{}{
				"test":  "test1",
				"test2": "random-value",
			},
			filterKey: "test",
			filterValues: &FilterSet{Set: map[interface{}]bool{
				"comcast": true,
				"sky":     true,
			}},
		},
		{
			description: "no key match",
			claims: map[string]interface{}{
				"test":  []interface{}{"test1", "random"},
				"test2": "random-value",
			},
			store: map[string]interface{}{
				"test":  "test1",
				"test2": "random-value",
			},
			filterKey: "random-key",
			filterValues: &FilterSet{Set: map[interface{}]bool{
				"test1":  true,
				"random": true,
			}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			m := new(device.Metadata)
			m.SetClaims(tc.claims)

			for key, val := range tc.store {
				m.Store(key, val)
			}

			fs := FilterStore(map[string]Set{
				tc.filterKey: tc.filterValues,
			})

			match, result := fs.metadataMatch(tc.filterKey, tc.filterValues, m)
			assert.Equal(tc.expectedMatch, match)
			assert.Equal(tc.expectedMatchResult, result)
		})

	}
}

func TestMarshalJSON(t *testing.T) {
	assert := assert.New(t)
	tests := []struct {
		description    string
		filterSet      *FilterSet
		expectedOutput []byte
	}{
		{
			description: "Successful String Unmarshal",
			filterSet: &FilterSet{Set: map[interface{}]bool{
				"test1": true,
				"test2": true,
			}},
			expectedOutput: []byte(`["test1","test2"]`),
		},
		{
			description: "Successful Int Unmarshal",
			filterSet: &FilterSet{Set: map[interface{}]bool{
				1: true,
				2: true,
				3: true,
			}},
			expectedOutput: []byte(`[1,2,3]`),
		},
		{
			description:    "Empty Set",
			filterSet:      &FilterSet{Set: map[interface{}]bool{}},
			expectedOutput: []byte(`[]`),
		},
		{
			description:    "Nil Set",
			filterSet:      nil,
			expectedOutput: []byte(`null`),
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			JSON, err := json.Marshal(tc.filterSet)
			fmt.Println(string(JSON))
			assert.ElementsMatch(tc.expectedOutput, JSON)
			assert.Nil(err)
		})

	}
}

package devicegate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/webpa-common/device"
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
		filters     map[string]Set
		canPass     bool
	}{
		{
			description: "Allow",
			canPass:     true,
			filters: map[string]Set{
				"partner-id": FilterSet(map[interface{}]bool{
					"comcast": true,
				}),
			},
		},
		{
			description: "Deny-Filter Match in Claims",
			canPass:     false,
			filters: map[string]Set{
				"partner-id": FilterSet(map[interface{}]bool{
					"comcast":        true,
					"random-partner": true,
				}),
			},
		},
		{
			description: "Deny-Filter Match in Metadata Store",
			canPass:     false,
			filters: map[string]Set{
				"random-key": FilterSet(map[interface{}]bool{
					"abc":    true,
					"random": true,
				}),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			mockDevice := new(device.MockDevice)

			mockDevice.On("Metadata").Return(metadata)

			fg := FilterGate{
				FilterStore: FilterStore(tc.filters),
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
			expectedSet:   FilterSet(map[interface{}]bool{"test": true, "test1": true}),
			expectedFound: true,
		},
		{
			description:   "Update",
			keyToSet:      "test",
			valuesToSet:   []interface{}{"random-value"},
			keyToGet:      "test",
			expectedSet:   FilterSet(map[interface{}]bool{"random-value": true}),
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
		description string
		filterGate  FilterGate
		setExists   bool
	}{
		{
			description: "Non-empty allowed filters set",
			filterGate: FilterGate{
				AllowedFilters: FilterSet(map[interface{}]bool{
					"test":          true,
					"random-filter": true,
				}),
			},
			setExists: true,
		},
		{
			description: "Empty allowed filters set",
			filterGate: FilterGate{
				AllowedFilters: FilterSet(map[interface{}]bool{}),
			},
			setExists: true,
		},
		{
			description: "Nil allowed filters set",
			filterGate:  FilterGate{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			filters, isSet := tc.filterGate.GetAllowedFilters()

			assert.Equal(tc.setExists, isSet)

			if tc.setExists {
				assert.NotNil(filters)
			} else {
				assert.Nil(filters)
			}
		})
	}
}

package key

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCachePeriodString(t *testing.T) {
	var testData = []struct {
		period         CachePeriod
		expectedString string
	}{
		{CachePeriodDefault, "default"},
		{CachePeriodForever, "forever"},
		{CachePeriodNever, "never"},
		{CachePeriod(24 * time.Hour), "24h0m0s"},
		{CachePeriod(30 * time.Minute), "30m0s"},
		{CachePeriod(-123), "forever"},
	}

	for _, test := range testData {
		actualString := test.period.String()
		if actualString != test.expectedString {
			t.Errorf("Expected String() [%s] but got [%s]", test.expectedString, actualString)
		}
	}
}

func TestCachePeriodMarshalJSON(t *testing.T) {
	var testData = []struct {
		period       CachePeriod
		expectedJSON string
	}{
		{CachePeriodDefault, `"default"`},
		{CachePeriodForever, `"forever"`},
		{CachePeriodNever, `"never"`},
		{CachePeriod(24 * time.Hour), `"24h0m0s"`},
		{CachePeriod(30 * time.Minute), `"30m0s"`},
		{CachePeriod(-123), `"forever"`},
	}

	for _, test := range testData {
		actualJSON, err := json.Marshal(test.period)
		if err != nil {
			t.Fatalf("Failed to marshal JSON: %v", err)
		}

		if string(actualJSON) != test.expectedJSON {
			t.Errorf("Expected JSON [%s] but got [%s]", test.expectedJSON, actualJSON)
		}
	}
}

func TestCachePeriodUnmarshalJSON(t *testing.T) {
	var validTestData = []struct {
		JSON           string
		expectedPeriod CachePeriod
	}{
		{`"default"`, CachePeriodDefault},
		{`"never"`, CachePeriodNever},
		{`"forever"`, CachePeriodForever},
		{`"24h"`, CachePeriod(24 * time.Hour)},
		{`"24h0m0s"`, CachePeriod(24 * time.Hour)},
		{`"30m"`, CachePeriod(30 * time.Minute)},
		{`"30m0s"`, CachePeriod(30 * time.Minute)},
	}

	var invalidTestData = []string{
		"",
		"0",
		"123",
		`""`,
		`"invalid"`,
		`"-30m"`,
		`"FOREVER"`,
		`"NeVeR"`,
	}

	for _, test := range validTestData {
		var actualPeriod CachePeriod
		err := json.Unmarshal([]byte(test.JSON), &actualPeriod)
		if err != nil {
			t.Fatalf("Failed to unmarshal JSON: %v", err)
		}

		if actualPeriod != test.expectedPeriod {
			t.Errorf("Expected period [%d] but got [%d]", test.expectedPeriod, actualPeriod)
		}
	}

	for _, invalidJSON := range invalidTestData {
		var actualPeriod CachePeriod
		err := json.Unmarshal([]byte(invalidJSON), &actualPeriod)
		if err == nil {
			t.Errorf("Should have failed to marshal JSON [%s]", invalidJSON)
		}
	}
}

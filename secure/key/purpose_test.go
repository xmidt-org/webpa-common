package key

import (
	"encoding/json"
	"testing"
)

func TestPurposeString(t *testing.T) {
	var testData = []struct {
		purpose        Purpose
		expectedString string
	}{
		{PurposeSign, "sign"},
		{PurposeVerify, "verify"},
		{PurposeEncrypt, "encrypt"},
		{PurposeDecrypt, "decrypt"},
		{Purpose(45), "verify"},
		{Purpose(-1), "verify"},
	}

	for _, test := range testData {
		actualString := test.purpose.String()
		if actualString != test.expectedString {
			t.Errorf("Expected String() [%s] but got [%s]", test.expectedString, actualString)
		}
	}
}

func TestPurposeMarshalJSON(t *testing.T) {
	var testData = []struct {
		purpose      Purpose
		expectedJSON string
	}{
		{PurposeSign, `"sign"`},
		{PurposeVerify, `"verify"`},
		{PurposeEncrypt, `"encrypt"`},
		{PurposeDecrypt, `"decrypt"`},
		{Purpose(45), `"verify"`},
		{Purpose(-1), `"verify"`},
	}

	for _, test := range testData {
		actualJSON, err := json.Marshal(test.purpose)
		if err != nil {
			t.Fatalf("Failed to marshal JSON: %v", err)
		}

		if string(actualJSON) != test.expectedJSON {
			t.Errorf("Expected JSON [%s] but got [%s]", test.expectedJSON, actualJSON)
		}
	}
}

func TestPurposeUnmarshalJSON(t *testing.T) {
	var validRecords = []struct {
		JSON            string
		expectedPurpose Purpose
	}{
		{`"sign"`, PurposeSign},
		{`"verify"`, PurposeVerify},
		{`"encrypt"`, PurposeEncrypt},
		{`"decrypt"`, PurposeDecrypt},
	}

	var invalidRecords = []string{
		"",
		"0",
		"123",
		`""`,
		`"invalid"`,
		`"SIGN"`,
		`"vERifY"`,
	}

	for _, test := range validRecords {
		var actualPurpose Purpose
		err := json.Unmarshal([]byte(test.JSON), &actualPurpose)
		if err != nil {
			t.Fatalf("Failed to unmarshal JSON: %v", err)
		}

		if actualPurpose != test.expectedPurpose {
			t.Errorf("Expected purpose [%d] but got [%d]", test.expectedPurpose, actualPurpose)
		}
	}

	for _, invalidJSON := range invalidRecords {
		var actualPurpose Purpose
		err := json.Unmarshal([]byte(invalidJSON), &actualPurpose)
		if err == nil {
			t.Errorf("Should have failed to marshal JSON [%s]", invalidJSON)
		}
	}
}

package device

import (
	"testing"
)

func TestParseID(t *testing.T) {
	testData := []struct {
		id       string
		expected string
		valid    bool
	}{
		{"MAC:11:22:33:44:55:66", "mac:112233445566", true},
		{"MAC:11aaBB445566", "mac:11aabb445566", true},
		{"mac:11-aa-BB-44-55-66", "mac:11aabb445566", true},
		{"mac:11,aa,BB,44,55,66", "mac:11aabb445566", true},
		{"uuid:anything Goes!", "uuid:anything Goes!", true},
		{"dns:anything Goes!", "dns:anything Goes!", true},
		{"serial:1234", "serial:1234", true},
		{"mac:11-aa-BB-44-55-66/service", "mac:11aabb445566/service/", true},
		{"mac:11-aa-BB-44-55-66/service/", "mac:11aabb445566/service/", true},
		{"mac:11-aa-BB-44-55-66/service/ignoreMe", "mac:11aabb445566/service/", true},
		{"mac:11-aa-BB-44-55-66/service/foo/bar", "mac:11aabb445566/service/", true},
		{"invalid:a-BB-44-55", "", false},
		{"mac:11-aa-BB-44-55", "", false},
		{"MAC:invalid45566", "", false},
	}

	for _, record := range testData {
		id, err := ParseID(record.id)

		if err != nil {
			if record.valid {
				t.Errorf("Unexpected error for %s", record.id)
			}
		} else {
			if !record.valid {
				t.Fatalf("Expected error for %s", record.id)
			}

			if actual := string(id.Bytes()); actual != record.expected {
				t.Errorf("For %s, ParseId() returned %s, but was expecting %s", record.id, actual, record.expected)
			}
		}
	}
}

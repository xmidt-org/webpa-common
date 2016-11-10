package types

import (
	"fmt"
	"testing"
	"time"
)

var durationStrings = []struct {
	value    Duration
	expected string
}{
	{Duration(0), "0s"},
	{Duration(-1), "-1ns"},
	{Duration(10 * time.Second), "10s"},
	{Duration(-7 * time.Minute), "-7m0s"},
	{Duration(1500 * time.Millisecond), "1.5s"},
	{Duration(1 * time.Hour), "1h0m0s"},
}

func TestDurationStringer(t *testing.T) {
	for _, record := range durationStrings {
		actual := record.value.String()
		if record.expected != actual {
			t.Errorf("Expected %s, but got %s", record.expected, actual)
		}
	}
}

func TestDurationMarshalJSON(t *testing.T) {
	for _, record := range durationStrings {
		actual, err := record.value.MarshalJSON()
		if err != nil {
			t.Fatalf("Failed to marshal duration: %v", err)
		}

		expected := fmt.Sprintf(`"%s"`, record.expected)
		if expected != string(actual) {
			t.Errorf("Expected %s, but got %s", expected, actual)
		}
	}
}

func TestDurationUnmarshalJSON(t *testing.T) {
	for _, record := range durationStrings {
		// perform the reverse conversion
		jsonValue := fmt.Sprintf(`"%s"`, record.expected)
		var actual Duration
		if err := actual.UnmarshalJSON([]byte(jsonValue)); err != nil {
			t.Fatalf("Failed to unmarshal duration: %v", err)
		}

		if record.value != actual {
			t.Errorf("Expected %s, but got %s", record.value, actual)
		}
	}

	var integralValues = []struct {
		input    int
		expected Duration
	}{
		{1000, Duration(1000)},
		{1245798273, Duration(1245798273)},
		{0, Duration(0)},
		{-45, Duration(-45)},
	}

	for _, record := range integralValues {
		jsonValue := fmt.Sprintf("%d", record.input)
		var actual Duration
		if err := actual.UnmarshalJSON([]byte(jsonValue)); err != nil {
			t.Fatalf("Failed to unmarshal duration: %v", err)
		}

		if record.expected != actual {
			t.Errorf("Expected %s, but got %s", record.expected, actual)
		}
	}
}

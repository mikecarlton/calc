package main

import (
	"testing"
)

func TestParseTime(t *testing.T) {
	tests := []struct {
		input    string
		expected string // Expected result in seconds as string
		valid    bool   // Whether the input should be valid
	}{
		// Valid formats - seconds only
		{"45", "45", true},
		{"45.5", "45.5", true},
		{"0", "0", true},
		{"123.75", "123.75", true},

		// Valid formats - minutes:seconds
		{"1:30", "90", true},        // 1*60 + 30 = 90
		{"0:45", "45", true},        // 0*60 + 45 = 45
		{"2:15", "135", true},       // 2*60 + 15 = 135
		{"30:45.5", "1845.5", true}, // 30*60 + 45.5 = 1845.5
		{"5:30.25", "330.25", true}, // 5*60 + 30.25 = 330.25

		// Valid formats - hours:minutes:seconds
		{"1:30:45", "5445", true},     // 1*3600 + 30*60 + 45 = 5445
		{"0:0:30", "30", true},        // 0*3600 + 0*60 + 30 = 30
		{"2:15:30", "8130", true},     // 2*3600 + 15*60 + 30 = 8130
		{"1:30:45.5", "5445.5", true}, // 1*3600 + 30*60 + 45.5 = 5445.5
		{"10:0:0", "36000", true},     // 10*3600 = 36000

		// Invalid formats - fractional hours
		{"1.5:30:45", "", false},
		{"0.5:0:0", "", false},

		// Invalid formats - fractional minutes
		{"1:30.5:45", "", false},
		{"0:15.25:30", "", false},

		// Invalid formats - too many parts
		{"1:2:3:4", "", false},

		// Invalid formats - non-numeric parts
		{"abc:30:45", "", false},
		{"1:abc:45", "", false},
		{"1:30:abc", "", false},
		{"", "", false},

		// Invalid formats - negative values
		{"-1:30:45", "", false},
		{"1:-30:45", "", false},
		{"1:30:-45", "", false},

		// Edge cases
		{"0:0:0", "0", true},
		{"59:59.999", "3599.999", true}, // 59*60 + 59.999 = 3599.999
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result, valid := parseTime(test.input)

			if valid != test.valid {
				t.Errorf("parseTime(%q) validity = %v, want %v", test.input, valid, test.valid)
				return
			}

			if test.valid {
				if result == nil {
					t.Errorf("parseTime(%q) returned nil result for valid input", test.input)
					return
				}

				expectedNum := newNumber(test.expected)
				if result.String() != expectedNum.String() {
					t.Errorf("parseTime(%q) = %v, want %v", test.input, result.String(), expectedNum.String())
				}
			} else {
				if result != nil {
					t.Errorf("parseTime(%q) returned non-nil result for invalid input: %v", test.input, result.String())
				}
			}
		})
	}
}

// Test that parseTime integrates correctly with the number parsing system
func TestParseTimeIntegration(t *testing.T) {
	// Test that time values can be used in calculations
	time1, ok1 := parseTime("1:30") // 90 seconds
	if !ok1 {
		t.Fatal("Failed to parse time 1:30")
	}

	time2, ok2 := parseTime("0:30") // 30 seconds
	if !ok2 {
		t.Fatal("Failed to parse time 0:30")
	}

	// Add the times
	result := add(time1, time2) // Should be 120 seconds
	expected := newNumber("120")

	if result.String() != expected.String() {
		t.Errorf("Adding times: got %v, want %v", result.String(), expected.String())
	}
}

// Test edge cases for integral validation
func TestParseTimeIntegralValidation(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		// These should be valid (integral hours/minutes)
		{"1:30:45", true},
		{"0:0:45.5", true},
		{"10:59:0", true},

		// These should be invalid (fractional hours/minutes)
		{"1.0:30:45", false}, // Even 1.0 is considered fractional in our implementation
		{"1:30.0:45", false}, // Even 30.0 is considered fractional
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			_, valid := parseTime(test.input)
			if valid != test.valid {
				t.Errorf("parseTime(%q) validity = %v, want %v", test.input, valid, test.valid)
			}
		})
	}
}

// Test the helper function isNonNegativeInteger
func TestIsNonNegativeInteger(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"123", true},
		{"0", true},
		{"00", true},
		{"1.0", false},
		{"1.5", false},
		{"-1", false},
		{"abc", false},
		{"", false},
		{"12a", false},
		{"a12", false},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := isNonNegativeInteger(test.input)
			if result != test.expected {
				t.Errorf("isNonNegativeInteger(%q) = %v, want %v", test.input, result, test.expected)
			}
		})
	}
}
package main

import (
	"testing"
)

func TestParseTime(t *testing.T) {
	tests := []struct {
		input    string
		expected *Number
		valid    bool // Whether the input should be valid
	}{
		// Valid formats - minutes:seconds (returns fractional minutes)
		{"1:30", newNumber("1.5"), true},    // 1 + 30/60 = 1.5 minutes
		{"0:45", newNumber("0.75"), true},   // 0 + 45/60 = 0.75 minutes
		{"2:15", newNumber("2.25"), true},   // 2 + 15/60 = 2.25 minutes
		{"30:45", newNumber("30.75"), true}, // 30 + 45/60 = 30.75 minutes
		{"5:30", newNumber("5.5"), true},    // 5 + 30/60 = 5.5 minutes

		// Valid formats - hours:minutes:seconds (returns fractional hours)
		{"1:30:45", newNumber("1.5125"), true},              // 1 + 30/60 + 45/3600 = 1.5125 hours
		{"0:0:30", newNumber("0.008333333333333333"), true}, // 0 + 0/60 + 30/3600 = 0.008333... hours
		{"2:15:30", newNumber("2.258333333333333"), true},   // 2 + 15/60 + 30/3600 = 2.258333... hours

		// Invalid formats - fractional hours
		{"1.5:30:45", nil, false},
		{"0.5:0:0", nil, false},

		// Invalid formats - fractional minutes
		{"1:30.5:45", nil, false},
		{"0:15.25:30", nil, false},

		// Invalid formats - too many parts
		{"1:2:3:4", nil, false},

		// Invalid formats - non-numeric parts
		{"abc:30:45", nil, false},
		{"1:abc:45", nil, false},
		{"1:30:abc", nil, false},
		{"", nil, false},

		// Invalid formats - negative values
		{"-1:30:45", nil, false},
		{"1:-30:45", nil, false},
		{"1:30:-45", nil, false},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result, valid := parseBase60(test.input)

			if valid != test.valid {
				t.Errorf("parseBase60(%q) validity = %v, want %v", test.input, valid, test.valid)
				return
			}

			if test.valid {
				if result == nil {
					t.Errorf("parseBase60(%q) returned nil result for valid input", test.input)
					return
				}

				if result.String() != test.expected.String() {
					t.Errorf("parseBase60(%q) = %v, want %v", test.input, result.String(), test.expected.String())
				}
			} else {
				if result != nil {
					t.Errorf("parseBase60(%q) returned non-nil result for invalid input: %v", test.input, result.String())
				}
			}
		})
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
			_, valid := parseBase60(test.input)
			if valid != test.valid {
				t.Errorf("parseBase60(%q) validity = %v, want %v", test.input, valid, test.valid)
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

// Test binary magnitude parsing functionality
func TestBinaryMagnitudeParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		valid    bool
	}{
		// Basic magnitude tests
		{"1K", "1024", true},
		{"1M", "1048576", true},
		{"1G", "1073741824", true},
		{"1T", "1099511627776", true},
		{"1P", "1125899906842624", true},
		{"1E", "1152921504606846976", true},
		{"1Z", "1180591620717411303424", true},
		{"1Y", "1208925819614629174706176", true},

		// Multiple factors
		{"2K", "2048", true},
		{"3M", "3145728", true},
		{"10G", "10737418240", true},

		// Fractional base numbers
		{"1.5K", "1536", true},
		{"2.5M", "2621440", true},
		{"0.5G", "536870912", true},

		// Negative numbers
		{"-1K", "-1024", true},
		{"-2M", "-2097152", true},

		// Numbers without magnitude (should still work)
		{"1024", "1024", true},
		{"100", "100", true},
		{"42.5", "42.5", true},

		// Invalid magnitude suffixes (should parse as regular numbers)
		{"1X", "1", true}, // X is not in MAGNITUDE, so it stops at "1"
		{"1A", "1", true}, // A is not in MAGNITUDE

		// Edge cases
		{"0K", "0", true},
		{"0M", "0", true},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result, remainder := NewFromString(test.input)

			if test.valid {
				if result == nil {
					t.Errorf("NewFromString(%q) returned nil for valid input", test.input)
					return
				}

				expectedNum := newNumber(test.expected)
				if result.String() != expectedNum.String() {
					t.Errorf("NewFromString(%q) = %v, want %v", test.input, result.String(), expectedNum.String())
				}

				// For valid magnitude suffixes, remainder should be empty
				if test.input[len(test.input)-1:] == "K" || test.input[len(test.input)-1:] == "M" ||
					test.input[len(test.input)-1:] == "G" || test.input[len(test.input)-1:] == "T" ||
					test.input[len(test.input)-1:] == "P" || test.input[len(test.input)-1:] == "E" ||
					test.input[len(test.input)-1:] == "Z" || test.input[len(test.input)-1:] == "Y" {
					if remainder != "" {
						t.Errorf("NewFromString(%q) remainder = %q, want empty", test.input, remainder)
					}
				}
			} else {
				if result != nil {
					t.Errorf("NewFromString(%q) returned non-nil result for invalid input: %v", test.input, result.String())
				}
			}
		})
	}
}

// Test binary magnitude edge cases and error conditions
func TestBinaryMagnitudeEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		remainder string
	}{
		// Test that invalid suffixes don't interfere
		{"Invalid suffix X", "100X", "100", "X"},
		{"Invalid suffix A", "50A", "50", "A"},
		{"Invalid suffix lowercase k", "1k", "1", "k"}, // lowercase not supported

		// Test hex numbers with magnitude (should not apply magnitude to hex)
		{"Hex with K", "0x10K", "16384", ""},
		{"Binary with M", "0b1010M", "10485760", ""},

		// Test multiple magnitude letters (only first one should be used)
		{"Multiple K", "1KK", "1024", "K"}, // First K processed, second K is remainder

		// Test magnitude at beginning (invalid)
		{"K at start", "K100", "", "K100"}, // No number found

		// Test empty magnitude
		{"Empty string", "", "", ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, remainder := NewFromString(test.input)

			if test.expected == "" {
				if result != nil {
					t.Errorf("NewFromString(%q) = %v, want nil", test.input, result.String())
				}
			} else {
				if result == nil {
					t.Errorf("NewFromString(%q) returned nil, want %v", test.input, test.expected)
					return
				}

				expectedNum := newNumber(test.expected)
				if result.String() != expectedNum.String() {
					t.Errorf("NewFromString(%q) = %v, want %v", test.input, result.String(), expectedNum.String())
				}
			}

			if remainder != test.remainder {
				t.Errorf("NewFromString(%q) remainder = %q, want %q", test.input, remainder, test.remainder)
			}
		})
	}
}

// Test integration of binary magnitude with calculations
func TestBinaryMagnitudeCalculations(t *testing.T) {
	// Test that magnitude numbers work in arithmetic
	k1, _ := NewFromString("1K") // 1024
	k2, _ := NewFromString("2K") // 2048

	// Test addition: 1K + 2K = 3K = 3072
	sum := add(k1, k2)
	expected := newNumber("3072")
	if sum.String() != expected.String() {
		t.Errorf("1K + 2K = %v, want %v", sum.String(), expected.String())
	}

	// Test multiplication: 2 * 1M = 2M = 2097152
	m1, _ := NewFromString("1M") // 1048576
	two := newNumber("2")
	product := mul(two, m1)
	expected2 := newNumber("2097152")
	if product.String() != expected2.String() {
		t.Errorf("2 * 1M = %v, want %v", product.String(), expected2.String())
	}
}

// Test temperature addition rules
func TestTemperatureAddition(t *testing.T) {
	tests := []struct {
		name       string
		left       string
		leftUnit   string
		right      string
		rightUnit  string
		op         string
		expected   string
		shouldFail bool
	}{
		// Valid cases - same absolute units
		{"C + C", "20", "C", "10", "C", "+", "30 °C", false},
		{"F + F", "68", "F", "10", "F", "+", "78 °F", false},
		{"C - C", "30", "C", "10", "C", "-", "20 °C", false},
		{"F - F", "86", "F", "18", "F", "-", "68 °F", false},

		// Valid cases - delta + absolute (same scale)
		{"C + dC", "20", "C", "10", "dC", "+", "30 °C", false},
		{"F + dF", "68", "F", "18", "dF", "+", "86 °F", false},
		{"C - dC", "30", "C", "10", "dC", "-", "20 °C", false},
		{"F - dF", "86", "F", "18", "dF", "-", "68 °F", false},

		// Valid cases - delta + absolute (cross scale)
		{"C + dF", "20", "C", "18", "dF", "+", "30 °C", false}, // 18°FΔ = 10°CΔ
		{"F + dC", "68", "F", "10", "dC", "+", "86 °F", false}, // 10°CΔ = 18°FΔ
		{"C - dF", "30", "C", "18", "dF", "-", "20 °C", false},
		{"F - dC", "86", "F", "10", "dC", "-", "68 °F", false},

		// Valid cases - delta + delta
		{"dC + dC", "10", "dC", "5", "dC", "+", "15 °CΔ", false},
		{"dF + dF", "18", "dF", "9", "dF", "+", "27 °FΔ", false},
		{"dC + dF", "10", "dC", "18", "dF", "+", "20 °CΔ", false}, // 18°FΔ = 10°CΔ
		{"dF + dC", "18", "dF", "10", "dC", "+", "36 °FΔ", false}, // 10°CΔ = 18°FΔ

		// Invalid cases - different absolute units
		{"C + F invalid", "20", "C", "68", "F", "+", "", true},
		{"F + C invalid", "68", "F", "20", "C", "+", "", true},
		{"C - F invalid", "30", "C", "68", "F", "-", "", true},
		{"F - C invalid", "86", "F", "20", "C", "-", "", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create values with units
			leftVal := Value{
				number: newNumber(test.left),
				units:  createSingleUnit(test.leftUnit),
			}
			rightVal := Value{
				number: newNumber(test.right),
				units:  createSingleUnit(test.rightUnit),
			}

			if test.shouldFail {
				// Test should panic/fail
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Expected %s %s %s to fail, but it succeeded", test.left+test.leftUnit, test.op, test.right+test.rightUnit)
					}
				}()

				_ = leftVal.binaryOp(test.op, rightVal)
			} else {
				// Test should succeed
				result := leftVal.binaryOp(test.op, rightVal)
				if result.String() != test.expected {
					t.Errorf("%s%s %s %s%s = %s, want %s",
						test.left, test.leftUnit, test.op, test.right, test.rightUnit,
						result.String(), test.expected)
				}
			}
		})
	}
}

// Test temperature conversion functionality separately
func TestTemperatureConversion(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		fromUnit string
		toUnit   string
		expected string
	}{
		// Absolute temperature conversions
		{"32F to C", "32", "F", "C", "0 °C"},
		{"0C to F", "0", "C", "F", "32 °F"},
		{"100C to F", "100", "C", "F", "212 °F"},
		{"212F to C", "212", "F", "C", "100 °C"},
		{"-40F to C", "-40", "F", "C", "-40 °C"}, // Crossover point

		// Delta temperature conversions
		{"18dF to dC", "18", "dF", "dC", "10 °CΔ"},
		{"10dC to dF", "10", "dC", "dF", "18 °FΔ"},
		{"5dC to dC", "5", "dC", "dC", "5 °CΔ"}, // Same units
		{"9dF to dF", "9", "dF", "dF", "9 °FΔ"}, // Same units
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			val := Value{
				number: newNumber(test.value),
				units:  createSingleUnit(test.fromUnit),
			}

			targetUnit := createSingleUnit(test.toUnit)
			result := val.apply(targetUnit)

			if result.String() != test.expected {
				t.Errorf("%s %s to %s = %s, want %s",
					test.value, test.fromUnit, test.toUnit, result.String(), test.expected)
			}
		})
	}
}

// Helper function to create a Unit array with a single temperature unit
func createSingleUnit(unitName string) Unit {
	var units Unit
	if unitUnit, exists := UNITS[unitName]; exists {
		// Copy the Unit array from UNITS table
		for dim, unit := range unitUnit {
			if unit.power != 0 {
				units[dim] = unit
			}
		}
	}
	return units
}

// Test temperature edge cases and validation
func TestTemperatureEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		description string
		operation   func() interface{}
		shouldPanic bool
		expectValue string
	}{
		{
			name:        "Zero absolute addition",
			description: "0°C + 0°C should equal 0°C",
			operation: func() interface{} {
				left := Value{number: newNumber("0"), units: createSingleUnit("C")}
				right := Value{number: newNumber("0"), units: createSingleUnit("C")}
				return left.binaryOp("+", right)
			},
			shouldPanic: false,
			expectValue: "0 °C",
		},
		{
			name:        "Negative delta addition",
			description: "20°C + (-10°CΔ) should equal 10°C",
			operation: func() interface{} {
				left := Value{number: newNumber("20"), units: createSingleUnit("C")}
				right := Value{number: newNumber("-10"), units: createSingleUnit("dC")}
				return left.binaryOp("+", right)
			},
			shouldPanic: false,
			expectValue: "10 °C",
		},
		{
			name:        "Large temperature delta",
			description: "0°C + 100°CΔ should equal 100°C",
			operation: func() interface{} {
				left := Value{number: newNumber("0"), units: createSingleUnit("C")}
				right := Value{number: newNumber("100"), units: createSingleUnit("dC")}
				return left.binaryOp("+", right)
			},
			shouldPanic: false,
			expectValue: "100 °C",
		},
		{
			name:        "Temperature multiplication not allowed",
			description: "Temperature * Temperature should be invalid",
			operation: func() interface{} {
				left := Value{number: newNumber("20"), units: createSingleUnit("C")}
				right := Value{number: newNumber("68"), units: createSingleUnit("F")}
				return left.binaryOp("*", right)
			},
			shouldPanic: true,
			expectValue: "",
		},
		{
			name:        "Same temperature multiplication not allowed",
			description: "°C * °C should be invalid",
			operation: func() interface{} {
				left := Value{number: newNumber("20"), units: createSingleUnit("C")}
				right := Value{number: newNumber("30"), units: createSingleUnit("C")}
				return left.binaryOp("*", right)
			},
			shouldPanic: true,
			expectValue: "",
		},
		{
			name:        "Scalar multiplication allowed",
			description: "2 * 20°C should equal 40°C",
			operation: func() interface{} {
				left := Value{number: newNumber("2"), units: Unit{}}
				right := Value{number: newNumber("20"), units: createSingleUnit("C")}
				return left.binaryOp("*", right)
			},
			shouldPanic: false,
			expectValue: "40 °C",
		},
		{
			name:        "Temperature scalar multiplication allowed",
			description: "20°C * 2 should equal 40°C",
			operation: func() interface{} {
				left := Value{number: newNumber("20"), units: createSingleUnit("C")}
				right := Value{number: newNumber("2"), units: Unit{}}
				return left.binaryOp("*", right)
			},
			shouldPanic: false,
			expectValue: "40 °C",
		},
		{
			name:        "Division allows different absolute units",
			description: "Temperature division should work regardless of units",
			operation: func() interface{} {
				left := Value{number: newNumber("100"), units: createSingleUnit("C")}
				right := Value{number: newNumber("50"), units: createSingleUnit("F")}
				return left.binaryOp("/", right)
			},
			shouldPanic: false,
			expectValue: "2",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.shouldPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Expected operation to panic, but it succeeded")
					}
				}()
				test.operation()
			} else {
				result := test.operation()
				if val, ok := result.(Value); ok {
					if val.String() != test.expectValue {
						t.Errorf("%s: got %s, want %s", test.description, val.String(), test.expectValue)
					}
				} else {
					t.Errorf("Expected Value result, got %T", result)
				}
			}
		})
	}
}

// Test negative number formatting in different bases
func TestNegativeNumberFormatting(t *testing.T) {
	tests := []struct {
		name     string
		number   string
		base     int
		expected string
	}{
		// Hexadecimal
		{"Negative hex -16", "-16", 16, "-0x10"},
		{"Negative hex -255", "-255", 16, "-0xff"},
		{"Negative hex -10", "-10", 16, "-0xa"},
		{"Positive hex 16", "16", 16, "0x10"},

		// Binary
		{"Negative binary -8", "-8", 2, "-0b1000"},
		{"Negative binary -15", "-15", 2, "-0b1111"},
		{"Negative binary -1", "-1", 2, "-0b1"},
		{"Positive binary 8", "8", 2, "0b1000"},

		// Octal
		{"Negative octal -8", "-8", 8, "-0o10"},
		{"Negative octal -64", "-64", 8, "-0o100"},
		{"Negative octal -7", "-7", 8, "-0o7"},
		{"Positive octal 8", "8", 8, "0o10"},

		// Edge cases
		{"Zero hex", "0", 16, "0x0"},
		{"Zero binary", "0", 2, "0b0"},
		{"Zero octal", "0", 8, "0o0"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			num := newNumber(test.number)
			result := toString(num, test.base)

			if result != test.expected {
				t.Errorf("toString(%s, %d) = %s, want %s", test.number, test.base, result, test.expected)
			}
		})
	}
}

// Test negative number formatting with fractional numbers (should fall back to decimal)
func TestNegativeNumberFormattingFractional(t *testing.T) {
	tests := []struct {
		name     string
		number   string
		base     int
		expected string // Should be decimal representation
	}{
		{"Negative fractional hex", "-16.5", 16, "-16.5"},
		{"Negative fractional binary", "-8.25", 2, "-8.25"},
		{"Negative fractional octal", "-7.125", 8, "-7.125"},
		{"Positive fractional hex", "16.5", 16, "16.5"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			num := newNumber(test.number)
			result := toString(num, test.base)

			if result != test.expected {
				t.Errorf("toString(%s, %d) = %s, want %s", test.number, test.base, result, test.expected)
			}
		})
	}
}

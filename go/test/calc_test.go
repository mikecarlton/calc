// Copyright 2024 Mike Carlton
// Released under terms of the MIT License:
//   http://www.opensource.org/licenses/mit-license.php

//go:build integration
// +build integration

package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func runCalc(args ...string) (string, error) {
	// Build the calculator in the test directory from parent source
	buildCmd := exec.Command("go", "build", "-o", "calc", "..")
	if err := buildCmd.Run(); err != nil {
		return "", err
	}

	// Run the calculator with the given arguments
	cmd := exec.Command("./calc", args...)
	output, err := cmd.Output()
	return strings.TrimSpace(string(output)), err
}

func TestDefaultPrecision(t *testing.T) {
	// Test that 1/3 shows 4 digits of precision by default
	output, err := runCalc("1", "3", "/")
	if err != nil {
		t.Fatalf("Error running calc: %v", err)
	}

	expected := "0.3333"
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestTimeParsingIntegration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"seconds only", "45.5", "45.5"},
		{"minutes and seconds", "1:30", "90"},
		{"hours minutes seconds", "1:30:45", "5445"},
		{"fractional seconds", "1:30:45.5", "5445.5"},
		{"zero values", "0:0:0", "0"},
		{"large values", "10:59:59.999", "39599.999"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, err := runCalc(test.input)
			if err != nil {
				t.Fatalf("Error running calc with %q: %v", test.input, err)
			}

			if output != test.expected {
				t.Errorf("Time parsing %q: expected %q, got %q", test.input, test.expected, output)
			}
		})
	}
}

func TestTimeParsingInvalidFormats(t *testing.T) {
	invalidInputs := []string{
		"1.5:30:45",   // fractional hours
		"1:30.5:45",   // fractional minutes
		"-1:30:45",    // negative hours
		"1:-30:45",    // negative minutes
		"1:30:-45",    // negative seconds
		"1:2:3:4",     // too many parts
		"abc:30:45",   // non-numeric hours
		"1:abc:45",    // non-numeric minutes
		"1:30:abc",    // non-numeric seconds
	}

	for _, input := range invalidInputs {
		t.Run(input, func(t *testing.T) {
			// For invalid inputs, we expect the command to exit with non-zero status
			cmd := exec.Command("./calc", input)
			cmd.Dir = "."
			// Build first
			buildCmd := exec.Command("go", "build", "-o", "calc", "..")
			buildCmd.Dir = "."
			if err := buildCmd.Run(); err != nil {
				t.Fatalf("Failed to build calc: %v", err)
			}
			
			output, err := cmd.CombinedOutput() // Get both stdout and stderr
			if err == nil {
				t.Errorf("Expected error for invalid time format %q, but got none", input)
			}
			// The error output should contain "Unrecognized argument"
			outputStr := string(output)
			if !strings.Contains(outputStr, "Unrecognized argument") {
				t.Errorf("Expected 'Unrecognized argument' in output for %q, got: %q", input, outputStr)
			}
		})
	}
}

func TestTimeWithUnitConversions(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{"time to minutes", []string{"1:30", "s", "min"}, "1.5 min"},
		{"time to hours", []string{"1:30:00", "s", "hr"}, "1.5 hr"},
		{"complex time to minutes", []string{"1:30:45", "s", "min"}, "90.75 min"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, err := runCalc(test.args...)
			if err != nil {
				t.Fatalf("Error running calc with %v: %v", test.args, err)
			}

			if output != test.expected {
				t.Errorf("Time conversion %v: expected %q, got %q", test.args, test.expected, output)
			}
		})
	}
}

func TestCustomPrecision(t *testing.T) {
	// Test that -p flag overrides default precision
	testCases := []struct {
		precision string
		expected  string
	}{
		{"2", "0.33"},
		{"6", "0.333333"},
		{"8", "0.33333333"},
	}

	for _, tc := range testCases {
		output, err := runCalc("-p", tc.precision, "1", "3", "/")
		if err != nil {
			t.Fatalf("Error running calc with precision %s: %v", tc.precision, err)
		}

		if output != tc.expected {
			t.Errorf("With precision %s, expected %q, got %q", tc.precision, tc.expected, output)
		}
	}
}

func TestPrecisionWithOtherOperations(t *testing.T) {
	// Test that precision works with other operations too
	output, err := runCalc("2", "3", "/")
	if err != nil {
		t.Fatalf("Error running calc: %v", err)
	}

	expected := "0.6667"
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestPrecisionDoesNotAffectIntegers(t *testing.T) {
	// Test that precision doesn't affect integer results
	output, err := runCalc("6", "3", "/")
	if err != nil {
		t.Fatalf("Error running calc: %v", err)
	}

	expected := "2"
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestPiConstant(t *testing.T) {
	// Test that pi constant is pushed to stack
	output, err := runCalc("pi")
	if err != nil {
		t.Fatalf("Error running calc with pi: %v", err)
	}

	// Pi should start with 3.1415 (first 4 decimal places with default precision)
	expected := "3.1416" // Pi rounded to 4 decimal places
	if output != expected {
		t.Errorf("Expected pi to be %q, got %q", expected, output)
	}
}

func TestPiInCalculation(t *testing.T) {
	// Test that pi works in calculations
	output, err := runCalc("pi", "2", "*")
	if err != nil {
		t.Fatalf("Error running calc with pi * 2: %v", err)
	}

	// 2 * pi should be approximately 6.2832
	expected := "6.2832" // 2*Pi rounded to 4 decimal places
	if output != expected {
		t.Errorf("Expected 2*pi to be %q, got %q", expected, output)
	}
}

func TestBinaryNumbers(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"0b101", "5"},   // 5 in binary
		{"0b1111", "15"}, // 15 in binary
		{"0b1000", "8"},  // 8 in binary
		{"0B110", "6"},   // Test uppercase B
	}

	for _, tc := range testCases {
		output, err := runCalc(tc.input)
		if err != nil {
			t.Fatalf("Error running calc with %s: %v", tc.input, err)
		}

		if output != tc.expected {
			t.Errorf("For %s, expected %q, got %q", tc.input, tc.expected, output)
		}
	}
}

func TestHexNumbers(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"0x1F", "31"},
		{"0xA", "10"},
		{"0xFF", "255"},
		{"0X10", "16"},     // uppercase X
		{"0x10.8", "16.5"}, // hex float
		{"0x1p-2", "0.25"}, // the power is bits, not hex
		{"0x10p2", "64"},
	}

	for _, tc := range testCases {
		output, err := runCalc(tc.input)
		if err != nil {
			t.Fatalf("Error running calc with %s: %v", tc.input, err)
		}

		if output != tc.expected {
			t.Errorf("For %s, expected %q, got %q", tc.input, tc.expected, output)
		}
	}
}

func TestBinaryHexCalculations(t *testing.T) {
	// Test calculations mixing binary, hex, and decimal
	output, err := runCalc("0b101", "0x1F", "+")
	if err != nil {
		t.Fatalf("Error running calc with binary + hex: %v", err)
	}

	expected := "36" // 5 + 31 = 36
	if output != expected {
		t.Errorf("Expected binary + hex to be %q, got %q", expected, output)
	}
}

func TestCleanup(t *testing.T) {
	// Clean up the test binary after tests
	os.Remove("./calc")
}

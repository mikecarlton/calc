// Copyright 2024 Mike Carlton
// Released under terms of the MIT License:
//   http://www.opensource.org/licenses/mit-license.php

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

func TestCleanup(t *testing.T) {
	// Clean up the test binary after tests
	os.Remove("./calc")
}
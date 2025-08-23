// Copyright 2024 Mike Carlton
// Released under terms of the MIT License:
//   http://www.opensource.org/licenses/mit-license.php

package main

import (
	"fmt"
)

type Value struct {
	number *Number
	units  Unit
}

type Operator struct {
	exec           NumericOp
	multiplicative bool
	unary          bool
	dimensionless  bool
	integerOnly    bool
}

var OPALIAS = Aliases{
	".":   "*",
	"•":   "*",
	"pow": "**",
}

var OPERATOR = map[string]Operator{
	"+":     {exec: add},
	"-":     {exec: sub},
	"*":     {exec: mul, multiplicative: true},
	"/":     {exec: div, multiplicative: true},
	"%":     {exec: mod, dimensionless: true},
	"**":    {exec: pow, multiplicative: true, dimensionless: true},
	"chs":   {exec: neg, unary: true},
	"t":     {exec: truncate, unary: true},
	"r":     {exec: reciprocal, multiplicative: true, unary: true},
	"log":   {exec: log, dimensionless: true, unary: true},
	"log10": {exec: log10, dimensionless: true, unary: true},
	"log2":  {exec: log2, dimensionless: true, unary: true},
	"rand":  {exec: random, dimensionless: true, unary: true},
	"mask":  {exec: mask, dimensionless: true, unary: true, integerOnly: true},

	// Bitwise operations (integers only)
	"&":  {exec: bitwiseAnd, dimensionless: true, integerOnly: true},
	"|":  {exec: bitwiseOr, dimensionless: true, integerOnly: true},
	"^":  {exec: bitwiseXor, dimensionless: true, integerOnly: true},
	"<<": {exec: leftShift, dimensionless: true, integerOnly: true},
	">>": {exec: rightShift, dimensionless: true, integerOnly: true},
	"~":  {exec: bitwiseNot, dimensionless: true, integerOnly: true, unary: true},
}

func (v Value) binaryOp(op string, other Value) Value {
	if OPERATOR[op].integerOnly && (!v.number.isIntegral() || !other.number.isIntegral()) {
		panic(fmt.Sprintf("Integer values required for '%s'", op))
	}

	if OPERATOR[op].dimensionless && !other.units.empty() {
		panic(fmt.Sprintf("Dimensionless value required for '%s', got '%s'", op, other))
	}

	if OPERATOR[op].multiplicative {
		// For multiplication/division with temperatures, check special rules
		if (op == "*" || op == "**" || op == "pow") && !temperatureMultiplicationValid(v.units, other.units) {
			panic(fmt.Sprintf("Invalid temperature operation: cannot multiply temperatures %s %s %s", v.units, op, other.units))
		}
		other = other.convertTo(v.units)
		v = unitBinaryOp(op, v, other)
	} else {
		if v.units.compatible(other.units) {
			// For addition/subtraction with temperatures, check special rules
			if (op == "+" || op == "-") && !temperatureAdditionValid(v.units, other.units) {
				panic(fmt.Sprintf("Invalid temperature operation: %s %s %s", v.units, op, other.units))
			}
		} else {
			panic(fmt.Sprintf("Incompatible units for '%s': %s vs %s", op, v.units.Name(), other.units.Name()))
		}
		other = other.convertTo(v.units)
	}

	v.number = OPERATOR[op].exec(v.number, other.number)
	return v
}

func (v Value) unaryOp(op string) Value {
	if OPERATOR[op].integerOnly && !v.number.isIntegral() {
		panic(fmt.Sprintf("Integer value required for '%s'", op))
	}
	if OPERATOR[op].dimensionless && !v.units.empty() {
		panic(fmt.Sprintf("Dimensionless-value required for '%s', got '%s'", op, v))
	} else if OPERATOR[op].multiplicative {
		v = unitUnaryOp(op, v)
	}

	v.number = OPERATOR[op].exec(v.number, nil)
	return v
}

func abs(n int) int {
	if n < 0 {
		return -n
	} else {
		return n
	}
}

// converts v to units
// when adding or subtracting, there must first be a check that the units are compatible (i.e. same power on all dimensions)
// when multiplying or dividing, units are converted to the new units
// will never remove units from value
func (v Value) convertTo(units Unit) Value {
	if options.debug {
		fmt.Printf("(%s).convert(%s) -->", green(v.String()), green(units.String()))
	}

	for dim, unit := range units {
		if unit.power == 0 || v.units[dim].power == 0 {
			// do nothing
		} else {
			factor := div(v.units[dim].factor, units[dim].factor)
			if v.units[dim].factor != nil && unit.factor != nil {
				v.number = mul(v.number, intPow(factor, v.units[dim].power))
				v.units[dim].BaseUnit = unit.BaseUnit
			} else {
				panic(fmt.Sprintf("Incomplete for %s -> %s", v.units[dim].name, unit.name))
				// At least one unit uses dynamic conversion
				if unit.factorFunction != nil {
					v.number = unit.factorFunction(v.number, v.units[dim].BaseUnit, unit.BaseUnit)
				} else if v.units[dim].factorFunction != nil {
					v.number = v.units[dim].factorFunction(v.number, v.units[dim].BaseUnit, unit.BaseUnit)
				} else {
					panic(fmt.Sprintf("No conversion method available for %s -> %s", v.units[dim].name, unit.name))
				}
			}
		}
	}

	if options.debug {
		fmt.Printf(" %s\n", green(v.String()))
	}
	return v
}

func (v Value) apply(units Unit) Value {
	if options.debug {
		fmt.Printf("(%s).apply(%s) -->", green(v.String()), green(units.String()))
	}

	if v.units.empty() || units.empty() {
		v.units = units
	} else if v.units.compatible(units) {
		for i, unit := range units {
			if unit.power == 0 || (unit.name == v.units[i].name && unit.power == v.units[i].power) {
				continue
			}
			// Use factor for simple scaling, or factorFunction for dynamic conversion
			if v.units[i].factor != nil && unit.factor != nil {
				// Both units use static factors - standard scaling conversion
				vFactor := intPow(v.units[i].factor, abs(unit.power))
				unitsFactor := intPow(unit.factor, abs(unit.power))
				if unit.power > 0 {
					v.number = div(mul(v.number, vFactor), unitsFactor)
				} else if unit.power < 0 {
					v.number = div(mul(v.number, unitsFactor), vFactor)
				}
			} else {
				// At least one unit uses dynamic conversion
				if unit.factorFunction != nil {
					v.number = unit.factorFunction(v.number, v.units[i].BaseUnit, unit.BaseUnit)
				} else if v.units[i].factorFunction != nil {
					v.number = v.units[i].factorFunction(v.number, v.units[i].BaseUnit, unit.BaseUnit)
				} else {
					panic(fmt.Sprintf("No conversion method available for %s -> %s", v.units[i].name, unit.name))
				}
			}
		}
		v.units = units
	} else {
		panic(fmt.Sprintf("Incompatible units %s vs %s", v.units, units))
	}

	if options.debug {
		fmt.Printf(" %s\n", green(v.String()))
	}

	return v
}

func (v Value) String() string {
	// Check if this is a time unit that should be displayed in time format
	if v.units[Time].power == 1 && v.isOnlyTimeUnit() {
		if v.units[Time].name == "hr" {
			return v.formatAsHours()
		} else if v.units[Time].name == "min" {
			return v.formatAsMinutes()
		}
	}

	var result string
	if options.showRational {
		result = fmt.Sprintf("%s (%d/%d)", v.number.String(), v.number.Num(), v.number.Denom())
	} else {
		result = v.number.String()
	}
	units := v.units.String()

	if units != "" {
		result += " " + units
	}
	return result
}

// isOnlyTimeUnit checks if this value only has time units (no other dimensions)
func (v Value) isOnlyTimeUnit() bool {
	for i, unit := range v.units {
		if i == int(Time) {
			continue // Skip time dimension
		}
		if unit.power != 0 {
			return false
		}
	}
	return true
}

// formatAsHours formats time value in hr units as H:MM:SS
func (v Value) formatAsHours() string {
	// Convert to seconds for calculation
	totalSecondsNum := mul(v.number, newNumber(3600))
	totalSeconds, _ := totalSecondsNum.Rat.Float64()

	hours := int(totalSeconds) / 3600
	minutes := (int(totalSeconds) % 3600) / 60
	seconds := int(totalSeconds) % 60

	// Handle fractional seconds
	fractionalSeconds := totalSeconds - float64(int(totalSeconds))
	if fractionalSeconds > 0 {
		return fmt.Sprintf("%d:%02d:%02d%s hr", hours, minutes, seconds, formatFraction(fractionalSeconds))
	}
	return fmt.Sprintf("%d:%02d:%02d hr", hours, minutes, seconds)
}

// formatAsMinutes formats time value in mn units as M:SS
func (v Value) formatAsMinutes() string {
	// Convert to seconds for calculation
	totalSecondsNum := mul(v.number, newNumber(60))
	totalSeconds, _ := totalSecondsNum.Rat.Float64()

	minutes := int(totalSeconds) / 60
	seconds := int(totalSeconds) % 60

	// Handle fractional seconds
	fractionalSeconds := totalSeconds - float64(int(totalSeconds))
	if fractionalSeconds > 0 {
		return fmt.Sprintf("%d:%02d%s min", minutes, seconds, formatFraction(fractionalSeconds))
	}
	return fmt.Sprintf("%d:%02d min", minutes, seconds)
}

// formatTimeAsHours formats just the time number part in hr units as H:MM:SS (no units suffix)
func (v Value) formatTimeAsHours() string {
	// Convert to seconds for calculation
	totalSecondsNum := mul(v.number, newNumber(3600))
	totalSeconds, _ := totalSecondsNum.Rat.Float64()

	hours := int(totalSeconds) / 3600
	minutes := (int(totalSeconds) % 3600) / 60
	seconds := int(totalSeconds) % 60

	// Handle fractional seconds
	fractionalSeconds := totalSeconds - float64(int(totalSeconds))
	if fractionalSeconds > 0 {
		return fmt.Sprintf("%d:%02d:%02d%s", hours, minutes, seconds, formatFraction(fractionalSeconds))
	}
	return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
}

// formatTimeAsMinutes formats just the time number part in min units as M:SS (no units suffix)
func (v Value) formatTimeAsMinutes() string {
	// Convert to seconds for calculation
	totalSecondsNum := mul(v.number, newNumber(60))
	totalSeconds, _ := totalSecondsNum.Rat.Float64()

	minutes := int(totalSeconds) / 60
	seconds := int(totalSeconds) % 60

	// Handle fractional seconds
	fractionalSeconds := totalSeconds - float64(int(totalSeconds))
	if fractionalSeconds > 0 {
		return fmt.Sprintf("%d:%02d%s", minutes, seconds, formatFraction(fractionalSeconds))
	}
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

// formatFraction formats fractional part of seconds (e.g., ".25" for 0.25)
func formatFraction(frac float64) string {
	if frac == 0 {
		return ""
	}
	// Format with appropriate precision, removing leading zero
	formatted := fmt.Sprintf("%.2f", frac)
	if formatted[0] == '0' {
		return formatted[1:] // Remove leading '0' to get just ".xx"
	}
	return formatted
}

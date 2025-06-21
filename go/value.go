// Copyright 2024 Mike Carlton
// Released under terms of the MIT License:
//   http://www.opensource.org/licenses/mit-license.php

package main

import (
	"fmt"
)

type Value struct {
	number *Number
	units  Units
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
	"r":     {exec: reciprocal, unary: true, multiplicative: true},
	"log":   {exec: log, dimensionless: true, unary: true},
	"log10": {exec: log10, dimensionless: true, unary: true},
	"log2":  {exec: log2, dimensionless: true, unary: true},

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
	} else if OPERATOR[op].multiplicative {
		// For multiplication/division with temperatures, check special rules
		if (op == "*" || op == "**" || op == "pow") && !temperatureMultiplicationValid(v.units, other.units) {
			panic(fmt.Sprintf("Invalid temperature operation: cannot multiply temperatures %s %s %s", v.units, op, other.units))
		}
		v = unitBinaryOp(op, v, other)
	} else {
		if v.units.compatible(other.units) {
			// For addition/subtraction with temperatures, check special rules
			if (op == "+" || op == "-") && !temperatureAdditionValid(v.units, other.units) {
				panic(fmt.Sprintf("Invalid temperature operation: %s %s %s", v.units, op, other.units))
			}
			other = other.apply(v.units)
		} else {
			panic(fmt.Sprintf("Incompatible units for '%s': %s vs %s", op, v.units.Name(), other.units.Name()))
		}
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

func (v Value) apply(units Units) Value {
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
					v.number = unit.factorFunction(v.number, v.units[i].UnitDef, unit.UnitDef)
				} else if v.units[i].factorFunction != nil {
					v.number = v.units[i].factorFunction(v.number, v.units[i].UnitDef, unit.UnitDef)
				} else {
					panic(fmt.Sprintf("No conversion method available for %s -> %s", v.units[i].name, unit.name))
				}
			}
		}
		v.units = units
	} else {
		panic(fmt.Sprintf("Incompatible units %s vs %s", v.units, units))
	}

	return v
}

func (v Value) String() string {
	result := v.number.String()
	units := v.units.String()

	if units != "" {
		result += " " + units
	}
	return result
}

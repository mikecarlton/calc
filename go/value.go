// Copyright 2024 Mike Carlton
// Released under terms of the MIT License:
//   http://www.opensource.org/licenses/mit-license.php

package main

import (
	"fmt"
)

type Value struct {
	number Number
	units  Units
}

type Operator struct {
	exec           NumericOp
	multiplicative bool
	unary          bool
	dimensionless  bool
	integerOnly    bool
}

var OPERATOR = map[string]Operator{
	"+":     {exec: add, multiplicative: false, unary: false, dimensionless: false},
	"-":     {exec: sub, multiplicative: false, unary: false, dimensionless: false},
	"*":     {exec: mul, multiplicative: true, unary: false, dimensionless: false},
	".":     {exec: mul, multiplicative: true, unary: false, dimensionless: false},
	"•":     {exec: mul, multiplicative: true, unary: false, dimensionless: false},
	"/":     {exec: div, multiplicative: true, unary: false, dimensionless: false},
	"chs":   {exec: neg, multiplicative: false, unary: true, dimensionless: false},
	"log":   {exec: log, multiplicative: false, unary: true, dimensionless: true},
	"log10": {exec: log10, multiplicative: false, unary: true, dimensionless: true},
	"log2":  {exec: log2, multiplicative: false, unary: true, dimensionless: true},
	"**":    {exec: pow, multiplicative: false, unary: false, dimensionless: true},
	"pow":   {exec: pow, multiplicative: false, unary: false, dimensionless: true},
}

func (v Value) binaryOp(op string, other Value) Value {
	if OPERATOR[op].multiplicative {
		v.units = unitBinaryOp(v.units, other.units, op)
	} else {
		if v.units.compatible(other.units) {
			other = other.apply(v.units)
		} else {
			panic(fmt.Sprintf("Incompatible units for '%s': %s vs %s", op, v.units, other.units))
		}
	}

	v.number = OPERATOR[op].exec(v.number, other.number)
	return v
}

func (v Value) unaryOp(op string) Value {
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
			if unit.power == 0 || unit == v.units[i] {
				continue
			}
			if i == int(Temperature) && units[i].UnitDef == UNITS["C"] { // F -> C
				v.number = sub(v.number, newInt(32))
			}
			vFactor := intPow(v.units[i].factor, abs(unit.power))
			unitsFactor := intPow(unit.factor, abs(unit.power))
			if unit.power > 0 {
				v.number = div(mul(v.number, vFactor), unitsFactor)
			} else if unit.power < 0 {
				v.number = div(mul(v.number, unitsFactor), vFactor)
			}
			if i == int(Temperature) && units[i].UnitDef == UNITS["F"] {
				v.number = add(v.number, newInt(32))
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
		result += " " + v.units.String()
	}
	return result
}

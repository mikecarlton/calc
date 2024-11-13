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

var OPALIAS = Aliases{
	".":   "*",
	"•":   "*",
	"pow": "**",
}

var OPERATOR = map[string]Operator{
	"+":     {exec: Number.Add},
	"-":     {exec: Number.Sub},
	"*":     {exec: Number.Mul, multiplicative: true},
	"/":     {exec: Number.Div, multiplicative: true},
	"**":    {exec: Number.Pow, multiplicative: true, dimensionless: true},
	"chs":   {exec: Number.Neg, unary: true},
	"t":     {exec: Number.Truncate, unary: true},
	"r":     {exec: Number.Reciprocal, unary: true, multiplicative: true},
	"log":   {exec: Number.Log, dimensionless: true, unary: true},
	"log10": {exec: Number.Log10, dimensionless: true, unary: true},
	"log2":  {exec: Number.Log2, dimensionless: true, unary: true},
}

func (v Value) binaryOp(op string, other Value) Value {
	if OPERATOR[op].dimensionless && !other.units.empty() {
		panic(fmt.Sprintf("Dimensionless value required for '%s', got '%s'", op, other))
	} else if OPERATOR[op].multiplicative {
		v = unitBinaryOp(op, v, other)
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
	if OPERATOR[op].dimensionless && !v.units.empty() {
		panic(fmt.Sprintf("Dimensionless-value required for '%s', got '%s'", op, v))
	} else if OPERATOR[op].multiplicative {
		v = unitUnaryOp(op, v)
	}

	v.number = OPERATOR[op].exec(v.number, Number{})
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
				if !v.units[i].delta && !units[i].delta {
					v.number = v.number.Sub(newNumber(32))
				}
			}
			vFactor := v.units[i].factor.IntPow(abs(unit.power))
			unitsFactor := unit.factor.IntPow(abs(unit.power))
			if unit.power > 0 {
				v.number = v.number.Mul(vFactor).Div(unitsFactor)
			} else if unit.power < 0 {
				v.number = v.number.Mul(unitsFactor).Div(vFactor)
			}
			if i == int(Temperature) && units[i].UnitDef == UNITS["F"] {
				if !v.units[i].delta && !units[i].delta {
					v.number = v.number.Add(newNumber(32))
				}
			}
		}
		v.units = units
	} else {
		panic(fmt.Sprintf("Incompatible units %s vs %s", v.units, units))
	}

	return v
}

/*
func (v Value) String() string {
	result := v.number.String()
	units := v.units.String()

	if units != "" {
		result += " " + units
	}
	return result
}
*/

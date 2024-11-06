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

func (v Value) binaryOp(other Value, op string) Value {
	if multiplicativeOp(op) {
		v.units = unitBinaryOp(v.units, other.units, op)
	} else {
		if v.units.compatible(other.units) {
			other = other.apply(v.units)
		} else {
			panic(fmt.Sprintf("Incompatible units for '%s': %s vs %s", op, v.units, other.units))
		}
	}
	v.number = numericBinaryOp(v.number, other.number, op)
	return v
}

func (v Value) unaryOp(op string) Value {
	v.number = numericUnaryOp(v.number, op)
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
			if unit.power == 0 {
				continue
			}
			vFactor := pow(v.units[i].factor, abs(unit.power))
			unitsFactor := pow(unit.factor, abs(unit.power))
			if unit.power > 0 {
				v.number = div(mul(v.number, vFactor), unitsFactor)
			} else if unit.power < 0 {
				v.number = div(mul(v.number, unitsFactor), vFactor)
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

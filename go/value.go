// Copyright 2024 Mike Carlton
// Released under terms of the MIT License:
//   http://www.opensource.org/licenses/mit-license.php

package main

type Value struct {
	number Number
	units  []Unit
}

func (v Value) binaryOp(other Value, op string) Value {
	v.number = v.number.binaryOp(other.number, op)

	return v
}

func (v Value) unaryOp(op string) Value {
	v.number = v.number.unaryOp(op)

	return v
}

func (v Value) String() string {
	return v.number.String() + " " + v.units.String()
}

// Copyright 2024 Mike Carlton
// Released under terms of the MIT License:
//   http://www.opensource.org/licenses/mit-license.php

package main

import (
	"math/big"
)

// Number is either a big integer or float
type Number struct {
	i *big.Int
	f *big.Float
}

func parseNumber(input string) (Number, bool) {
	if i, ok := new(big.Int).SetString(input, 10); ok {
		return Number{i: i}, true
	} else if f, ok := new(big.Float).SetString(input); ok {
		return Number{f: f}, true
	} else {
		return Number{}, false
	}
}

func (n Number) isInt() bool {
	if n.i != nil {
		return true
	} else {
		return false
	}
}

func (n Number) String() string {
	if n.i != nil {
		return n.i.String()
	}
	if n.f != nil {
		return n.f.String()
	}
	return ""
}

func (n Number) binaryOp(other Number, op string) Number {
	var left, right Number

	if n.isInt() && other.isInt() || !n.isInt() && !other.isInt() {
		left = n
		right = other
	} else if n.isInt() {
		left = Number{f: new(big.Float).SetInt(n.i)}
		right = other
	} else {
		left = n
		right = Number{f: new(big.Float).SetInt(other.i)}
	}

	switch op {
	case "+":
		if left.isInt() {
			left.i.Add(left.i, right.i)
		} else {
			left.f.Add(left.f, right.f)
		}
	case "-":
		if left.isInt() {
			left.i.Sub(left.i, right.i)
		} else {
			left.f.Sub(left.f, right.f)
		}
	case "*":
		if left.isInt() {
			left.i.Mul(left.i, right.i)
		} else {
			left.f.Mul(left.f, right.f)
		}
	case "/":
		if left.isInt() {
			var modulus big.Int
			original := new(big.Int).Set(left.i)
			left.i.DivMod(left.i, right.i, &modulus)
			if modulus.Sign() != 0 {
				f1 := new(big.Float).SetInt(original)
				f2 := new(big.Float).SetInt(right.i)
				left = Number{f: f1.Quo(f1, f2)}
			}
		} else {
			left.f.Quo(left.f, right.f)
		}
	}

	return left
}

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

const PRECISION = 113 // match IEEE 754 quadruple-precision binary floating-point format (binary128)

func newFloat() *big.Float {
	return new(big.Float).SetPrec(PRECISION)
}

func newInt() *big.Int {
	return new(big.Int)
}

func parseNumber(input string) (Number, bool) {
	if i, ok := newInt().SetString(input, 10); ok {
		return Number{i: i}, true
	} else if f, ok := newFloat().SetString(input); ok {
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

func cast(left, right Number) (Number, Number) {
	if left.isInt() && right.isInt() || !left.isInt() && !right.isInt() {
		return left, right
	} else if left.isInt() {
		return Number{f: newFloat().SetInt(left.i)}, right
	} else {
		return left, Number{f: newFloat().SetInt(right.i)}
	}
}

func (n Number) binaryOp2(other Number, op func(Number, Number) Number) Number {
	left, right := cast(n, other)
	if left.isInt() {
		left.i.op(left.i, right.i)
	} else {
		left.f.op(left.f, right.f)
	}

	return left
}

func (n Number) add(other Number) Number {
	return n.binaryOp2(other, big.Add)
}

func (n Number) add(other Number) Number {
	left, right := cast(n, other)
	if left.isInt() {
		left.i.Add(left.i, right.i)
	} else {
		left.f.Add(left.f, right.f)
	}

	return left
}

func (n Number) binaryOp(other Number, op string) Number {
	left, right := cast(n, other)
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
	case "*", ".":
		if left.isInt() {
			left.i.Mul(left.i, right.i)
		} else {
			left.f.Mul(left.f, right.f)
		}
	case "/":
		if left.isInt() {
			var modulus big.Int
			original := newInt().Set(left.i)
			left.i.DivMod(left.i, right.i, &modulus)
			if modulus.Sign() != 0 {
				f1 := newFloat().SetInt(original)
				f2 := newFloat().SetInt(right.i)
				left = Number{f: f1.Quo(f1, f2)}
			}
		} else {
			left.f.Quo(left.f, right.f)
		}
	}

	return left
}

func (n Number) unaryOp(op string) Number {
	switch op {
	case "chs":
		if n.isInt() {
			n.i.Neg(n.i)
		} else {
			n.f.Neg(n.f)
		}
	}

	return n
}

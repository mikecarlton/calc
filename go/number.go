// Copyright 2024 Mike Carlton
// Released under terms of the MIT License:
//   http://www.opensource.org/licenses/mit-license.php

package main

import (
	"fmt"
	"math/big"
)

// Number is either a *big.Int or *big.Float
type Number interface {
	fmt.Stringer
}

const PRECISION = 113 // match IEEE 754 quadruple-precision binary floating-point format (binary128)

func newFloat() *big.Float {
	return new(big.Float).SetPrec(PRECISION)
}

func newInt() *big.Int {
	return new(big.Int)
}

func isInt(n Number) bool {
	_, ok := n.(*big.Int)

	return ok
}

func parseNumber(input string) (Number, bool) {
	if i, ok := newInt().SetString(input, 10); ok {
		return i, true
	} else if f, ok := newFloat().SetString(input); ok {
		return f, true
	} else {
		return nil, false
	}
}

func cast(left, right Number) (Number, Number) {
	if leftTyped, ok := left.(*big.Int); ok {
		if _, ok := right.(*big.Int); ok { // both are *big.Int
			return left, right
		} else {
			return newFloat().SetInt(leftTyped), right // cast left to *big.Float
		}
	} else {
		if rightTyped, ok := right.(*big.Int); ok {
			return left, newFloat().SetInt(rightTyped) // cast right to *big.Float
		} else {
			return left, right // both are *big.Float
		}
	}
}

func add(left, right Number) Number {
	left, right = cast(left, right)

	if leftTyped, ok := left.(*big.Int); ok {
		rightTyped := right.(*big.Int)
		return leftTyped.Add(leftTyped, rightTyped)
	} else {
		leftTyped := left.(*big.Float)
		rightTyped := right.(*big.Float)
		return leftTyped.Add(leftTyped, rightTyped)
	}
}

func sub(left, right Number) Number {
	left, right = cast(left, right)

	if leftTyped, ok := left.(*big.Int); ok {
		rightTyped := right.(*big.Int)
		return leftTyped.Sub(leftTyped, rightTyped)
	} else {
		leftTyped := left.(*big.Float)
		rightTyped := right.(*big.Float)
		return leftTyped.Sub(leftTyped, rightTyped)
	}
}

func mul(left, right Number) Number {
	left, right = cast(left, right)

	if leftTyped, ok := left.(*big.Int); ok {
		rightTyped := right.(*big.Int)
		return leftTyped.Mul(leftTyped, rightTyped)
	} else {
		leftTyped := left.(*big.Float)
		rightTyped := right.(*big.Float)
		return leftTyped.Mul(leftTyped, rightTyped)
	}
}

func div(left, right Number) Number {
	left, right = cast(left, right)

	if leftTyped, ok := left.(*big.Int); ok {
		rightTyped := right.(*big.Int)

		var modulus big.Int
		result, _ := newInt().DivMod(leftTyped, rightTyped, &modulus)
		if modulus.Sign() == 0 {
			return result
		} else {
			f1 := newFloat().SetInt(leftTyped)
			f2 := newFloat().SetInt(rightTyped)
			return f1.Quo(f1, f2)
		}
	} else {
		leftTyped := left.(*big.Float)
		rightTyped := right.(*big.Float)
		return leftTyped.Quo(leftTyped, rightTyped)
	}
}

func neg(left Number) Number {
	if leftTyped, ok := left.(*big.Int); ok {
		return leftTyped.Neg(leftTyped)
	} else {
		leftTyped := left.(*big.Float)
		return leftTyped.Neg(leftTyped)
	}
}

func numericBinaryOp(left, right Number, op string) Number {
	left, right = cast(left, right)
	switch op {
	case "+":
		return add(left, right)
	case "-":
		return sub(left, right)
	case "*", ".":
		return mul(left, right)
	case "/":
		return div(left, right)
	default:
		panic(fmt.Sprintf("Unimplmented binary op: '%s'", op))
	}
}

func numericUnaryOp(n Number, op string) Number {
	switch op {
	case "chs":
		return neg(n)
	default:
		panic(fmt.Sprintf("Unimplmented unary op: '%s'", op))
	}
}

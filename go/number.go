// Copyright 2024 Mike Carlton
// Released under terms of the MIT License:
//   http://www.opensource.org/licenses/mit-license.php

package main

import (
	"fmt"
	"math"
	"math/big"
	"strings"
)

// Number is either a *big.Int or *big.Float
type Number interface {
	fmt.Stringer
}

type NumericOp func(Number, Number) Number

const PRECISION = 113 // match IEEE 754 quadruple-precision binary floating-point format (binary128)

func newFloat(vals ...float64) *big.Float {
	if len(vals) > 0 {
		return big.NewFloat(vals[0])
	}
	return new(big.Float)
}

var MAXINT = newInt(math.MaxInt)
var MININT = newInt(math.MinInt)

func newInt(vals ...int64) *big.Int {
	if len(vals) > 0 {
		return big.NewInt(vals[0])
	}
	return new(big.Int)
}

// returns Number stringified (with global precision if a float)
func toString(n Number) string {
	if _, ok := n.(*big.Int); ok {
		return n.String()
	} else {
		f := n.(*big.Float)
		if f.IsInt() {
			return fmt.Sprintf("%g", f)
		}
		return fmt.Sprintf("%.*f", options.precision, f)
	}
}

// returns Number as float64
// can lose precision or overflow to +Inf
func toFloat64(n Number) float64 {
	f64, _ := toFloat(n).Float64()
	return f64
}

// returns Number as Float
func toFloat(n Number) *big.Float {
	var float *big.Float
	if nTyped, ok := n.(*big.Int); ok {
		float = new(big.Float).SetInt(nTyped)
	} else {
		float = n.(*big.Float)
	}

	return float
}

func isInt(n Number) bool {
	_, ok := n.(*big.Int)

	return ok
}

// interpret numbers with ':' as base 60
func parseTime(input string) (Number, bool) {
	parts := strings.Split(input, ":")
	if len(parts) != 2 && len(parts) != 3 {
		return nil, false
	}

	var seconds Number
	var ok bool
	if seconds, ok = newInt().SetString(parts[len(parts)-1], 10); !ok {
		if seconds, ok = newFloat().SetString(parts[len(parts)-1]); !ok {
			return nil, false
		}
	}

	if minutes, ok := newInt().SetString(parts[len(parts)-2], 10); !ok {
		return nil, false
	} else {
		seconds = add(seconds, mul(minutes, newInt(60)))
	}

	if len(parts) == 3 {
		if hours, ok := newInt().SetString(parts[len(parts)-3], 10); !ok {
			return nil, false
		} else {
			seconds = add(seconds, mul(hours, newInt(3600)))
		}
	}

	return seconds, true
}

func parseNumber(input string) (Number, bool) {
	if i, ok := newInt().SetString(input, 0); ok {
		return i, true
	} else if f, ok := newFloat().SetString(input); ok {
		return f, true
	} else {
		return nil, false
	}
}

// returns left and right as *big.Int if both are *big.Int, else both as *big.Float
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

// raises left to power right, does not modify left inputs
func intPow(left Number, right int) Number {
	if right < 0 {
		panic(fmt.Sprintf("intPow is not defined for negative integers: '%d'", right))
	}

	exponent := int64(right)
	if leftTyped, ok := left.(*big.Int); ok {
		return newInt().Exp(leftTyped, newInt(exponent), nil)
	} else { // Exp is not defined on Float, do exponentiation by squaring
		leftTyped := left.(*big.Float)
		base := newFloat().Set(leftTyped)
		result := newFloat(1.0)
		for exponent > 0 {
			if exponent&1 == 1 {
				result.Mul(result, base)
			}
			base.Mul(base, base)
			exponent >>= 1
		}
		return result
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

func reciprocal(left, _ Number) Number {
	float := toFloat(left)
	return div(newFloat(1.0), float)
}

func truncate(left, _ Number) Number {
	if leftTyped, ok := left.(*big.Int); ok {
		return leftTyped
	} else {
		leftTyped := left.(*big.Float)
		leftInt, _ := leftTyped.Int(nil)
		return leftInt
	}
}

func neg(left, _ Number) Number {
	if leftTyped, ok := left.(*big.Int); ok {
		return leftTyped.Neg(leftTyped)
	} else {
		leftTyped := left.(*big.Float)
		return leftTyped.Neg(leftTyped)
	}
}

// Functions that resort to Math float64 (and so may lose precision)
type FloatBinaryOp func(float64, float64) float64
type FloatUnaryOp func(float64) float64

func doFloatBinary(op FloatBinaryOp, left, right Number) Number {
	return newFloat(op(toFloat64(left), toFloat64(right)))
}

func doFloatUnary(op FloatUnaryOp, left Number) Number {
	return newFloat(op(toFloat64(left)))
}

func log(left, _ Number) Number {
	return doFloatUnary(math.Log, left)
}

func log2(left, _ Number) Number {
	return doFloatUnary(math.Log2, left)
}

func log10(left, _ Number) Number {
	return doFloatUnary(math.Log10, left)
}

func pow(left, right Number) Number {
	if rightTyped, ok := right.(*big.Int); ok {
		i := int(rightTyped.Int64())
		if rightTyped.Cmp(MAXINT) < 1 && i >= 0 {
			return intPow(left, i)
		} else {
			return doFloatBinary(math.Pow, left, right)
		}
	} else {
		return doFloatBinary(math.Pow, left, right)
	}
}

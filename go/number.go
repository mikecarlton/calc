// Copyright 2024 Mike Carlton
// Released under terms of the MIT License:
//   http://www.opensource.org/licenses/mit-license.php

package main

import (
	"fmt"
	"math"
	"math/big"
)

type Element interface {
	fmt.Stringer
}

// Number is either a *big.Int or *big.Float
type Number struct {
	Element
}

type NumericOp func(Number, Number) Number

const PRECISION = 113 // match IEEE 754 quadruple-precision binary floating-point format (binary128)

func newNumber(val any) Number {
	switch val := val.(type) {
	case int:
		return Number{big.NewInt(int64(val))}
	case int64:
		return Number{big.NewInt(val)}
	case float64:
		return Number{big.NewFloat(val).SetPrec(PRECISION)}
	case Number:
		switch elt := val.Element.(type) {
		case *big.Int:
			return Number{new(big.Int).Set(elt)}
		case *big.Float:
			return Number{new(big.Float).Copy(elt)}
		default:
			panic(fmt.Sprintf("Unimplemented type '%T' in newNumber(Number)", elt))
		}
	default:
		panic(fmt.Sprintf("Unimplemented type '%T' in newNumber", val))
	}
}

// Number cast to *big.Float
func (x Number) Float() *big.Float {
	if xFloat, ok := x.Element.(*big.Float); ok {
		return xFloat
	} else {
		return new(big.Float).SetPrec(PRECISION).SetInt(x.Element.(*big.Int))
	}
}

// Number cast to float64
func (x Number) Float64() float64 {
	xFloat, _ := x.Float().Float64()
	return xFloat
}

var prefix = map[int]string{
	2:  "0b",
	8:  "0o",
	10: "",
	16: "0x",
}

// returns Number stringified (with global precision if a float)
func (x Number) String(base int) string {
	if xInt, ok := x.Element.(*big.Int); ok {
		if base == 60 {
			/*
				hours := newInt()
				minutes := newInt()
				seconds := newInt()
				hours.DivMod(nTyped, newInt(3600), seconds)
				minutes.DivMod(seconds, newInt(60), seconds)
				if hours.Int64() == 0 {
					return fmt.Sprintf("%d:%02d", minutes.Int64(), seconds.Int64())
				} else {
					return fmt.Sprintf("%d:%02d:%02d", hours.Int64(), minutes.Int64(), seconds.Int64())
				}
			*/
			return ""
		} else {
			return prefix[base] + xInt.Text(base)
		}
	} else {
		xFloat := x.Element.(*big.Float)
		if xFloat.IsInt() {
			i, _ := xFloat.Int64()
			return fmt.Sprintf("%d", i)
		}
		return fmt.Sprintf("%.*f", options.precision, xFloat)
	}
}

// interpret numbers with ':' as base 60
func parseTime(input string) (Number, bool) {
	/* TODO
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
	*/
	return newNumber(0), false
}

func parseNumber(input string) (Number, bool) {
	if i, ok := new(big.Int).SetString(input, 0); ok {
		return Number{i}, true
	} else if f, ok := new(big.Float).SetPrec(PRECISION).SetString(input); ok {
		return Number{f}, true
	} else {
		return Number{}, false
	}
}

func (x Number) Add(y Number) Number {
	xInt, xIsInt := x.Element.(*big.Int)
	yInt, yIsInt := y.Element.(*big.Int)

	if xIsInt && yIsInt {
		return Number{new(big.Int).Add(xInt, yInt)}
	}

	return Number{new(big.Float).Add(x.Float(), y.Float())}
}

func (x Number) Sub(y Number) Number {
	xInt, xIsInt := x.Element.(*big.Int)
	yInt, yIsInt := y.Element.(*big.Int)

	if xIsInt && yIsInt {
		return Number{new(big.Int).Sub(xInt, yInt)}
	}

	return Number{new(big.Float).Sub(x.Float(), y.Float())}
}

func (x Number) Mul(y Number) Number {
	xInt, xIsInt := x.Element.(*big.Int)
	yInt, yIsInt := y.Element.(*big.Int)

	if xIsInt && yIsInt {
		return Number{new(big.Int).Mul(xInt, yInt)}
	}

	return Number{new(big.Float).Mul(x.Float(), y.Float())}
}

func (x Number) Div(y Number) Number {
	xInt, xIsInt := x.Element.(*big.Int)
	yInt, yIsInt := y.Element.(*big.Int)

	if xIsInt && yIsInt {
		modulus := new(big.Int)
		result, _ := new(big.Int).DivMod(xInt, yInt, modulus)

		if modulus.Sign() == 0 {
			return Number{result}
		}
	}

	return Number{new(big.Float).Quo(x.Float(), y.Float())}
}

func (x Number) IntPow(y int) Number {
	if y < 0 {
		panic(fmt.Sprintf("IntPow is not defined for negative integers: '%d'", y))
	}

	exponent := int64(y)
	if xInt, xIsInt := x.Element.(*big.Int); xIsInt {
		return Number{new(big.Int).Exp(xInt, big.NewInt(exponent), nil)}
	} else { // Exp is not defined on Float, do exponentiation by squaring
		base := new(big.Float).Copy(x.Element.(*big.Float))
		result := big.NewFloat(1.0).SetPrec(PRECISION)
		for exponent > 0 {
			if exponent&1 == 1 {
				result.Mul(result, base)
			}
			base.Mul(base, base)
			exponent >>= 1
		}
		return Number{result}
	}
}

func (x Number) Reciprocal(_ Number) Number {
	one := big.NewFloat(1.0).SetPrec(PRECISION)
	return Number{one.Quo(one, x.Float())}
}

func (x Number) Truncate(_ Number) Number {
	n := new(big.Int)

	if xInt, xIsInt := x.Element.(*big.Int); xIsInt {
		return Number{n.Set(xInt)}
	} else {
		x.Float().Int(n) // TODO: does this modify x?
		return Number{n}
	}
}

func (x Number) Neg(_ Number) Number {
	if xInt, xIsInt := x.Element.(*big.Int); xIsInt {
		return Number{new(big.Int).Neg(xInt)}
	} else {
		return Number{new(big.Float).Neg(x.Float())}
	}
}

func (x Number) prec() uint {
	if xFloat, ok := x.Element.(*big.Float); ok {
		return xFloat.Prec()
	} else {
		return 0
	}
}

var MAXINT = big.NewInt(math.MaxInt)

func (x Number) Pow(y Number) Number {
	if yInt, yIsInt := y.Element.(*big.Int); yIsInt {
		i := int(yInt.Int64())
		if yInt.Cmp(MAXINT) < 1 && i >= 0 {
			return x.IntPow(i)
		}
	}

	return Number{big.NewFloat(math.Pow(x.Float64(), y.Float64())).SetPrec(PRECISION)}
}

func (x Number) Log(_ Number) Number {
	return Number{big.NewFloat(math.Log(x.Float64())).SetPrec(PRECISION)}
}

func (x Number) Log2(_ Number) Number {
	return Number{big.NewFloat(math.Log2(x.Float64())).SetPrec(PRECISION)}
}

func (x Number) Log10(_ Number) Number {
	return Number{big.NewFloat(math.Log10(x.Float64())).SetPrec(PRECISION)}
}

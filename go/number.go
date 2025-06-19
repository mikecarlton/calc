// Copyright 2024 Mike Carlton
// Released under terms of the MIT License:
//   http://www.opensource.org/licenses/mit-license.php

package main

import (
	"fmt"
	"math"
	"math/big"
	"regexp"
)

// Embed *big.Rat; all big.Rat methods can be applied directly on Number
type Number struct {
	*big.Rat
}

type NumericOp func(*Number, *Number) *Number

var PrecisionLimit int = 4 // default, overridden by options.precision
var Pi = NewNumber("3141592653589793238462643383279502884197/1000000000000000000000000000000000000000") // 40 digits ought to be enough

// stringifies a Number, with only as much precision (up to our configured limit) as is required to display exactly
func (n *Number) String() string {
	if n.Rat == nil {
		panic("Uninitialized Number")
	}

	precisionLimit := options.precision
	precision, exact := n.Rat.FloatPrec()
	if exact {
		precision = min(precisionLimit, precision)
	} else {
		precision = precisionLimit
	}
	return n.Rat.FloatString(precision)
}

func (n *Number) GoString() string { // for %#v format
	return fmt.Sprintf("%v {%v/%v}", n, n.Num(), n.Denom())
}

// parse Number from beginning of input, return *Number and remainder of the string
func NewFromString(input string) (*Number, string) {
	decimalPattern := `[+-]?(\d+(\.\d*)?|\.\d+)([eE][+-]?\d+)?`
	hexPattern := `[+-]?0[xX][0-9a-fA-F]+(\.[0-9a-fA-F]*)?[pP][+-]?\d+`
	binaryPattern := `[+-]?0[bB][01]+`
	pattern := fmt.Sprintf(`^(%s|%s|%s)`, decimalPattern, hexPattern, binaryPattern)
	re := regexp.MustCompile(pattern)

	match := re.FindString(input)
	if match == "" {
		return nil, input
	}

	return new(Number).Set(match), input[len(match):]
}

func NewNumber(value any) *Number {
	return new(Number).Set(value)
}

func (n *Number) Set(value any) *Number {
	if n.Rat == nil {
		n.Rat = new(big.Rat)
	}

	switch v := value.(type) {
	case int:
		n.SetInt64(int64(v))
	case uint:
		n.SetUint64(uint64(v))
	case int64:
		n.SetInt64(v)
	case uint64:
		n.SetUint64(v)
	case float64:
		n.SetFloat64(v)
	case string:
		_, ok := n.SetString(v)
		if !ok {
			panic(fmt.Sprintf("Invalid string: '%s'", v))
		}
	default:
		panic(fmt.Sprintf("Unsupported type: %T", v))
	}

	return n
}

func (n *Number) SetString(value string) (*Number, bool) {
	if n.Rat == nil {
		n.Rat = new(big.Rat)
	}
	_, ok := n.Rat.SetString(value)

	return n, ok
}

func (n *Number) Add(x, y *Number) *Number {
	if n.Rat == nil {
		n.Rat = new(big.Rat)
	}
	n.Rat.Add(x.Rat, y.Rat)

	return n
}

func (n *Number) Sub(x, y *Number) *Number {
	if n.Rat == nil {
		n.Rat = new(big.Rat)
	}
	n.Rat.Sub(x.Rat, y.Rat)

	return n
}

func (n *Number) Mul(x, y *Number) *Number {
	if n.Rat == nil { // Initialize if necessary
		n.Rat = new(big.Rat)
	}
	n.Rat.Mul(x.Rat, y.Rat)

	return n
}

func (n *Number) Quo(x, y *Number) *Number {
	if n.Rat == nil { // Initialize if necessary
		n.Rat = new(big.Rat)
	}
	n.Rat.Quo(x.Rat, y.Rat)

	return n
}

// Constants
const DOT = "·"

// Helper functions for arithmetic operations
func add(x, y *Number) *Number {
	result := new(Number)
	return result.Add(x, y)
}

func sub(x, y *Number) *Number {
	result := new(Number)
	return result.Sub(x, y)
}

func mul(x, y *Number) *Number {
	result := new(Number)
	return result.Mul(x, y)
}

func div(x, y *Number) *Number {
	result := new(Number)
	return result.Quo(x, y)
}

func pow(x, y *Number) *Number {
	// For now, implement simple integer power
	if y.Rat.IsInt() {
		exp := y.Rat.Num().Int64()
		result := NewNumber(1)
		base := NewNumber(x.String())
		
		if exp < 0 {
			base = reciprocal(base, nil)
			exp = -exp
		}
		
		for i := int64(0); i < exp; i++ {
			result = mul(result, base)
		}
		return result
	}
	
	// For non-integer powers, approximate using float64
	xFloat, _ := x.Rat.Float64()
	yFloat, _ := y.Rat.Float64()
	
	if xFloat <= 0 {
		panic("Cannot raise negative number to non-integer power")
	}
	
	result := math.Pow(xFloat, yFloat)
	return NewNumber(result)
}

func neg(x, y *Number) *Number {
	result := new(Number)
	result.Set(0)
	return result.Sub(result, x)
}

func truncate(x, y *Number) *Number {
	result := new(Number)
	result.Set(x.String())
	
	// Extract integer part
	intPart := new(big.Int)
	intPart.Quo(result.Rat.Num(), result.Rat.Denom())
	result.Rat.SetInt(intPart)
	
	return result
}

func reciprocal(x, y *Number) *Number {
	result := new(Number)
	one := NewNumber(1)
	return result.Quo(one, x)
}

func log(x, y *Number) *Number {
	xFloat, _ := x.Rat.Float64()
	if xFloat <= 0 {
		panic("Cannot take log of non-positive number")
	}
	
	result := math.Log(xFloat)
	return NewNumber(result)
}

func log10(x, y *Number) *Number {
	xFloat, _ := x.Rat.Float64()
	if xFloat <= 0 {
		panic("Cannot take log of non-positive number")
	}
	
	result := math.Log10(xFloat)
	return NewNumber(result)
}

func log2(x, y *Number) *Number {
	xFloat, _ := x.Rat.Float64()
	if xFloat <= 0 {
		panic("Cannot take log of non-positive number")
	}
	
	result := math.Log2(xFloat)
	return NewNumber(result)
}

// Helper functions
func newInt(value int) *Number {
	return NewNumber(value)
}

func toString(n *Number, base int) string {
	return n.String()
}

func intPow(base *Number, exp int) *Number {
	result := NewNumber(1)
	for i := 0; i < exp; i++ {
		result = mul(result, base)
	}
	return result
}

// Parsing functions
func parseNumber(input string) (*Number, bool) {
	num, remainder := NewFromString(input)
	if num != nil && remainder == "" {
		return num, true
	}
	return nil, false
}

func parseTime(input string) (*Number, bool) {
	// For now, just return false - time parsing not implemented yet
	return nil, false
}

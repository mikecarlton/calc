// Copyright 2024 Mike Carlton
// Released under terms of the MIT License:
//   http://www.opensource.org/licenses/mit-license.php

package main

import (
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
)

// Embed *big.Rat; all big.Rat methods can be applied directly on Number
type Number struct {
	*big.Rat
}

type NumericOp func(*Number, *Number) *Number

var PrecisionLimit int = 4                                                                              // default, overridden by options.precision
var Pi = newNumber("3141592653589793238462643383279502884197/1000000000000000000000000000000000000000") // 40 digits ought to be enough

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
	decimalPattern := `[+-]?(\d[\d,_]*(\.\d[\d,_]*)?|\.\d[\d,_]*)([eE][+-]?\d+)?`
	hexPattern := `[+-]?0[xX][0-9a-fA-F,_]+(\.[0-9a-fA-F,_]*)?([pP][+-]?\d+)?`
	binaryPattern := `[+-]?0[bB][01,_]+`
	magnitudePattern := fmt.Sprintf(`[%s]?`, MAGNITUDE)

	pattern := fmt.Sprintf(`^((%s)|(%s)|(%s))%s`, binaryPattern, hexPattern, decimalPattern, magnitudePattern)
	re := regexp.MustCompile(pattern)

	match := re.FindString(input)
	if match == "" {
		return nil, input
	}

	// Check for binary magnitude suffix
	if len(match) > 0 {
		lastChar := match[len(match)-1:]
		if strings.Contains(MAGNITUDE, lastChar) {
			// Extract the base number without the magnitude suffix
			baseStr := match[:len(match)-1]

			// Calculate binary factor: 2^((index+1) * 10)
			magnitudeIndex := strings.Index(MAGNITUDE, lastChar)
			exponent := (magnitudeIndex + 1) * 10

			// Use big.Int for very large factors to avoid overflow
			factor := new(big.Int)
			factor.Exp(big.NewInt(2), big.NewInt(int64(exponent)), nil)

			// Parse the base number and multiply by factor
			baseNum := new(Number).Set(baseStr)
			factorNum := new(Number)
			factorNum.Rat = new(big.Rat).SetInt(factor)
			result := mul(baseNum, factorNum)

			return result, input[len(match):]
		}
	}

	return new(Number).Set(match), input[len(match):]
}

func newNumber(value any) *Number {
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

	// If number contains comma or underscore separators, enable grouping
	if strings.Contains(value, ",") || strings.Contains(value, "_") {
		options.group = true
	}

	// Remove comma and underscore separators before parsing
	cleanValue := strings.ReplaceAll(strings.ReplaceAll(value, ",", ""), "_", "")
	_, ok := n.Rat.SetString(cleanValue)

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
const MAGNITUDE = "KMGTPEZY" // Binary magnitude suffixes

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
		result := newNumber(1)
		base := newNumber(x.String())

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
	return newNumber(result)
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
	one := newNumber(1)
	return result.Quo(one, x)
}

func log(x, y *Number) *Number {
	xFloat, _ := x.Rat.Float64()
	if xFloat <= 0 {
		panic("Cannot take log of non-positive number")
	}

	result := math.Log(xFloat)
	return newNumber(result)
}

func log10(x, y *Number) *Number {
	xFloat, _ := x.Rat.Float64()
	if xFloat <= 0 {
		panic("Cannot take log of non-positive number")
	}

	result := math.Log10(xFloat)
	return newNumber(result)
}

func log2(x, y *Number) *Number {
	xFloat, _ := x.Rat.Float64()
	if xFloat <= 0 {
		panic("Cannot take log of non-positive number")
	}

	result := math.Log2(xFloat)
	return newNumber(result)
}

func random(x, y *Number) *Number {
	return x.Mul(x, newNumber(rand.Float64()))
}

func sqrt(x, y *Number) *Number {
	xFloat, _ := x.Rat.Float64()
	if xFloat < 0 {
		panic("Cannot take square root of negative number")
	}

	result := math.Sqrt(xFloat)
	return newNumber(result)
}

// Bitwise operations - only work on integral numbers
func bitwiseAnd(x, y *Number) *Number {
	if !x.isIntegral() || !y.isIntegral() {
		panic("Bitwise operations require integral values")
	}

	xInt := new(big.Int)
	yInt := new(big.Int)
	xInt.Quo(x.Rat.Num(), x.Rat.Denom())
	yInt.Quo(y.Rat.Num(), y.Rat.Denom())

	result := new(big.Int)
	result.And(xInt, yInt)

	return newNumber(result.String())
}

func bitwiseOr(x, y *Number) *Number {
	if !x.isIntegral() || !y.isIntegral() {
		panic("Bitwise operations require integral values")
	}

	xInt := new(big.Int)
	yInt := new(big.Int)
	xInt.Quo(x.Rat.Num(), x.Rat.Denom())
	yInt.Quo(y.Rat.Num(), y.Rat.Denom())

	result := new(big.Int)
	result.Or(xInt, yInt)

	return newNumber(result.String())
}

func bitwiseXor(x, y *Number) *Number {
	if !x.isIntegral() || !y.isIntegral() {
		panic("Bitwise operations require integral values")
	}

	xInt := new(big.Int)
	yInt := new(big.Int)
	xInt.Quo(x.Rat.Num(), x.Rat.Denom())
	yInt.Quo(y.Rat.Num(), y.Rat.Denom())

	result := new(big.Int)
	result.Xor(xInt, yInt)

	return newNumber(result.String())
}

func leftShift(x, y *Number) *Number {
	if !x.isIntegral() || !y.isIntegral() {
		panic("Shift operations require integral values")
	}

	xInt := new(big.Int)
	yInt := new(big.Int)
	xInt.Quo(x.Rat.Num(), x.Rat.Denom())
	yInt.Quo(y.Rat.Num(), y.Rat.Denom())

	if !yInt.IsUint64() {
		panic("Shift amount must be a valid unsigned integer")
	}

	shift := yInt.Uint64()
	result := new(big.Int)
	result.Lsh(xInt, uint(shift))

	return newNumber(result.String())
}

func rightShift(x, y *Number) *Number {
	if !x.isIntegral() || !y.isIntegral() {
		panic("Shift operations require integral values")
	}

	xInt := new(big.Int)
	yInt := new(big.Int)
	xInt.Quo(x.Rat.Num(), x.Rat.Denom())
	yInt.Quo(y.Rat.Num(), y.Rat.Denom())

	if !yInt.IsUint64() {
		panic("Shift amount must be a valid unsigned integer")
	}

	shift := yInt.Uint64()
	result := new(big.Int)
	result.Rsh(xInt, uint(shift))

	return newNumber(result.String())
}

func bitwiseNot(x, y *Number) *Number {
	if !x.isIntegral() {
		panic("Bitwise operations require integral values")
	}

	xInt := new(big.Int)
	xInt.Quo(x.Rat.Num(), x.Rat.Denom())

	// For bitwise NOT, we XOR with 0xffffffffffffffff (64-bit mask)
	// This gives us simple bitwise inversion rather than 2's complement
	// TODO: we should XOR with mask of all 1s and same length as X
	mask := new(big.Int).SetUint64(math.MaxUint64)
	result := new(big.Int)
	result.Xor(xInt, mask)

	return newNumber(result.String())
}

// mask generates an IP mask with the specified number of bits
// e.g., mask(8) = 0xff000000, mask(24) = 0xffffff00
func mask(x, y *Number) *Number {
	if !x.isIntegral() {
		panic("Mask operation requires integral value")
	}

	bits := new(big.Int)
	bits.Quo(x.Rat.Num(), x.Rat.Denom())

	// Check if bits is in valid range (0-32)
	if bits.Sign() < 0 || bits.Cmp(big.NewInt(32)) > 0 {
		panic("Mask bits must be between 0 and 32")
	}

	// Create mask: shift left (32-bits) positions, then invert and shift left bits positions
	bitsInt := bits.Int64()

	if bitsInt == 0 {
		return newNumber(0)
	}
	if bitsInt == 32 {
		return newNumber("4294967295") // 0xffffffff
	}

	// Create mask by shifting 1s to the left
	// For n bits: (0xffffffff << (32-n)) & 0xffffffff
	result := new(big.Int)
	result.SetInt64(0xffffffff)
	result.Lsh(result, uint(32-bitsInt))
	result.And(result, big.NewInt(0xffffffff))

	return newNumber(result.String())
}

func mod(x, y *Number) *Number {
	if y.Rat.Sign() == 0 {
		panic("Division by zero in modulo operation")
	}

	// For rational numbers, compute x - y * floor(x/y)
	// This matches Ruby's behavior for modulo
	quotient := new(big.Rat)
	quotient.Quo(x.Rat, y.Rat)

	// Get the floor of the quotient
	floorInt := new(big.Int)
	floorInt.Quo(quotient.Num(), quotient.Denom())

	// If the quotient is negative and there's a remainder, subtract 1 to get floor
	remainder := new(big.Int)
	remainder.Rem(quotient.Num(), quotient.Denom())
	if quotient.Sign() < 0 && remainder.Sign() != 0 {
		floorInt.Sub(floorInt, big.NewInt(1))
	}

	floor := new(big.Rat)
	floor.SetInt(floorInt)

	// Calculate y * floor(x/y)
	product := new(big.Rat)
	product.Mul(y.Rat, floor)

	// Calculate x - y * floor(x/y)
	result := new(Number)
	result.Rat = new(big.Rat)
	result.Rat.Sub(x.Rat, product)

	return result
}

// newRationalNumber creates a Number from two int64 values (numerator/denominator)
func newRationalNumber(numerator, denominator int64) *Number {
	n := &Number{Rat: big.NewRat(numerator, denominator)}
	return n
}

// isIntegral returns true if the number represents an integer (denominator is 1)
func (n *Number) isIntegral() bool {
	return n.Rat.Denom().Cmp(big.NewInt(1)) == 0
}

// addCommaGrouping adds comma grouping to a decimal number string
func addCommaGrouping(s, separator string) string {
	// Handle negative numbers
	negative := false
	if strings.HasPrefix(s, "-") {
		negative = true
		s = s[1:]
	}

	// Split at decimal point
	parts := strings.Split(s, ".")
	integerPart := parts[0]

	// Add comma grouping to integer part (every 3 digits from right)
	if len(integerPart) > 3 {
		var result strings.Builder
		for i, digit := range integerPart {
			if i > 0 && (len(integerPart)-i)%3 == 0 {
				result.WriteString(separator)
			}
			result.WriteRune(digit)
		}
		integerPart = result.String()
	}

	// Reconstruct the number
	if len(parts) > 1 {
		integerPart += "." + parts[1]
	}

	if negative {
		return "-" + integerPart
	}
	return integerPart
}

// addUnderscoreGrouping adds underscore grouping to hex/binary/octal numbers (every 4 digits from right)
func addUnderscoreGrouping(s string) string {
	// Extract prefix and sign
	var prefix, sign, digits string

	if strings.HasPrefix(s, "-") {
		sign = "-"
		s = s[1:]
	}

	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		prefix = s[:2]
		digits = s[2:]
	} else if strings.HasPrefix(s, "0b") || strings.HasPrefix(s, "0B") {
		prefix = s[:2]
		digits = s[2:]
	} else if strings.HasPrefix(s, "0o") || strings.HasPrefix(s, "0O") {
		prefix = s[:2]
		digits = s[2:]
	} else {
		// No prefix, just digits
		digits = s
	}

	// Add underscore grouping (every 4 digits from right)
	if len(digits) > 4 {
		var result strings.Builder
		for i, digit := range digits {
			if i > 0 && (len(digits)-i)%4 == 0 {
				result.WriteString("_")
			}
			result.WriteRune(digit)
		}
		digits = result.String()
	}

	return sign + prefix + digits
}

// toFactor converts an integer to prime factorization format (e.g., "2 * 3^2 * 5")
func toFactor(n *Number) string {
	if !n.isIntegral() {
		return ""
	}

	xInt := new(big.Int)
	xInt.Quo(n.Rat.Num(), n.Rat.Denom())
	absX := new(big.Int).Abs(xInt)
	bigOne := big.NewInt(1)

	if xInt.Sign() == 0 || absX.Cmp(bigOne) == 0 { // Note that -1, 0 and 1 have no prime factorization
		return ""
	}

	// FactorPower represents a prime factor and its power
	type FactorPower struct {
		factor *big.Int
		power  int
	}

	factors := []FactorPower{}

	// simple trial division algorithm
	divisor := big.NewInt(2)
	for absX.Cmp(bigOne) > 0 {
		count := 0
		rem := new(big.Int)
		for rem.Rem(absX, divisor).Cmp(big.NewInt(0)) == 0 {
			count++
			absX.Div(absX, divisor)
		}

		if count > 0 {
			factors = append(factors, FactorPower{factor: new(big.Int).Set(divisor), power: count})
		}

		// Early break: if divisor² > absX, then absX is prime
		divisorSquared := new(big.Int).Mul(divisor, divisor)
		if divisorSquared.Cmp(absX) > 0 && absX.Cmp(bigOne) > 0 {
			factors = append(factors, FactorPower{factor: new(big.Int).Set(absX), power: 1})
			break
		}

		// Move to next divisor (2, then odd numbers)
		if divisor.Cmp(big.NewInt(2)) == 0 {
			divisor.Add(divisor, bigOne) // 2 -> 3
		} else {
			divisor.Add(divisor, big.NewInt(2)) // odd numbers
		}
	}

	// Format factorization string
	var parts []string
	if xInt.Sign() < 0 {
		parts = append(parts, "-1")
	}

	for _, fp := range factors {
		if fp.power == 1 {
			parts = append(parts, fp.factor.String())
		} else {
			parts = append(parts, fmt.Sprintf("%s^%d", fp.factor.String(), fp.power))
		}
	}

	return strings.Join(parts, " • ")
}

// toIPv4 converts an integer to IPv4 address format (e.g., "192.168.1.1")
func toIPv4(n *Number) string {
	if !n.isIntegral() {
		return ""
	}

	// Get the integer value
	intVal := new(big.Int)
	intVal.Quo(n.Rat.Num(), n.Rat.Denom())

	// Check if it's in valid IPv4 range (0 to 2^32-1)
	if intVal.Sign() < 0 {
		return ""
	}

	maxIPv4 := new(big.Int)
	maxIPv4.Lsh(big.NewInt(1), 32) // 2^32
	if intVal.Cmp(maxIPv4) >= 0 {
		return ""
	}

	// Convert to 4 bytes
	var octets [4]int64
	val := new(big.Int).Set(intVal)

	for i := 3; i >= 0; i-- {
		octet := new(big.Int)
		val.DivMod(val, big.NewInt(256), octet)
		octets[i] = octet.Int64()
	}

	return fmt.Sprintf("%d.%d.%d.%d", octets[0], octets[1], octets[2], octets[3])
}

func toString(n *Number, base int) string {
	if base == 10 {
		str := n.String()
		if options.group {
			return addCommaGrouping(str, ",")
		}
		return str
	}

	// For hexadecimal, support integral and optionally floating point
	if base == 16 {
		if n.isIntegral() {
			// Convert to integer for base conversion
			intVal := new(big.Int)
			intVal.Quo(n.Rat.Num(), n.Rat.Denom())

			var result string
			// Handle negative sign positioning
			if intVal.Sign() < 0 {
				intVal.Abs(intVal) // Make positive for formatting
				result = "-0x" + intVal.Text(16)
			} else {
				result = "0x" + intVal.Text(16)
			}

			// Add underscore grouping if -g option is enabled
			if options.group {
				result = addUnderscoreGrouping(result)
			}
			return result
		} else if options.showHexFloat {
			// Convert to float64 and format as hex floating point
			floatVal, _ := n.Rat.Float64()
			return strconv.FormatFloat(floatVal, 'x', -1, 64)
		} else {
			// Return decimal representation for non-integral numbers when hex float not enabled
			return n.String()
		}
	}

	// For binary and octal, we need the number to be integral
	if !n.isIntegral() {
		return n.String() // Return decimal representation for non-integral numbers
	}

	// Convert to integer for base conversion
	intVal := new(big.Int)
	intVal.Quo(n.Rat.Num(), n.Rat.Denom())

	// Handle negative sign positioning for binary and octal
	negative := intVal.Sign() < 0
	if negative {
		intVal.Abs(intVal) // Make positive for formatting
	}

	var result string
	switch base {
	case 2:
		result = "0b" + intVal.Text(2)
	case 8:
		result = "0o" + intVal.Text(8)
	default:
		result = intVal.Text(base)
	}

	if negative {
		result = "-" + result
	}

	// Add underscore grouping if -g option is enabled for binary and octal
	if options.group && (base == 2 || base == 8) {
		result = addUnderscoreGrouping(result)
	}

	return result
}

func intPow(base *Number, exp int) *Number {
	result := newNumber(1)
	if exp > 0 {
		for i := 0; i < exp; i++ {
			result = mul(result, base)
		}
	} else if exp < 0 {
		// Handle negative exponents by calculating the reciprocal
		baseReciprocal := reciprocal(base, nil)
		for i := 0; i < -exp; i++ {
			result = mul(result, baseReciprocal)
		}
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

// isNonNegativeInteger checks if a string represents a non-negative integer in decimal format
func isNonNegativeInteger(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// parseIPv4 parses an IPv4 address (e.g., "192.168.1.1") into a number
func parseIPv4(input string) (*Number, bool) {
	// Check if it matches IPv4 pattern: X.X.X.X where X is 0-255
	parts := strings.Split(input, ".")
	if len(parts) != 4 {
		return nil, false
	}

	result := newNumber(0)
	for _, part := range parts {
		// Parse each octet
		if octet, ok := parseNumber(part); ok && octet.isIntegral() {
			octetInt := new(big.Int)
			octetInt.Quo(octet.Rat.Num(), octet.Rat.Denom())

			// Check if it's in valid range (0-255)
			if octetInt.Sign() < 0 || octetInt.Cmp(big.NewInt(255)) > 0 {
				return nil, false
			}

			// Shift result left by 8 bits and add this octet
			result = mul(result, newNumber(256))
			result = add(result, octet)
		} else {
			return nil, false
		}
	}

	return result, true
}

func parseBase60(input string) (*Number, bool) {
	// Parse base-60 format: [hours:]minutes:seconds or minutes:seconds
	// All parts must be non-negative numbers, last part can be fractional
	// e.g. "1:30:45.5" or "30:45"

	parts := strings.Split(input, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return nil, false
	}

	var result *Number

	switch len(parts) {
	case 2:
		// Minutes:seconds format: "30:45.5"
		if !isNonNegativeInteger(parts[0]) {
			return nil, false
		}
		if sec, ok := parseNumber(parts[1]); ok && sec.Rat.Sign() >= 0 {
			min := newNumber(parts[0])
			// Convert to decimal: minutes + seconds/60
			secondsFrac := div(sec, newNumber(60))
			result = add(min, secondsFrac)
		} else {
			return nil, false
		}
	case 3:
		// Hours:minutes:seconds format: "1:30:45.5"
		if !isNonNegativeInteger(parts[0]) || !isNonNegativeInteger(parts[1]) {
			return nil, false
		}
		if sec, ok := parseNumber(parts[2]); ok && sec.Rat.Sign() >= 0 {
			hr := newNumber(parts[0])
			min := newNumber(parts[1])
			// Convert to decimal: hours + minutes/60 + seconds/3600
			minutesFrac := div(min, newNumber(60))
			secondsFrac := div(sec, newNumber(3600))
			result = add(add(hr, minutesFrac), secondsFrac)
		} else {
			return nil, false
		}
	}

	return result, true
}

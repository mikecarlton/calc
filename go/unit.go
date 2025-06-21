// Copyright 2024 Mike Carlton
// Released under terms of the MIT License:
//   http://www.opensource.org/licenses/mit-license.php

package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Dimension represents the type of unit (Length, Time, Mass, Temperature).
type Dimension int

const (
	Mass Dimension = iota
	Length
	Time
	Volume
	Temperature
	Currency
	NumDimension
)

type UnitDef struct {
	name           string
	description    string
	dimension      Dimension
	factor         *Number                                 // for simple scaling, nil for dynamic conversion
	delta          bool                                    // only applicable to Temperature
	factorFunction func(*Number, UnitDef, UnitDef) *Number // dynamic conversion function
}

type Unit struct {
	UnitDef
	power int
}

type Units [NumDimension]Unit

// currencyConvert handles any currency conversion, including multi-currency via USD
func currencyConvert(amount *Number, from, to UnitDef) *Number {
	fromCode, fromExists := getCurrencyCode(from.name)
	toCode, toExists := getCurrencyCode(to.name)
	
	if !fromExists || !toExists {
		panic(fmt.Sprintf("Unsupported currency conversion: %s -> %s", from.name, to.name))
	}
	
	var result *Number
	var err error
	
	// If either is USD, do direct conversion
	if fromCode == "USD" || toCode == "USD" {
		result, err = convertCurrency(amount, fromCode, toCode)
	} else {
		// Both are non-USD, convert through USD as intermediate
		// First convert from source to USD
		usdAmount, err1 := convertCurrency(amount, fromCode, "USD")
		if err1 != nil {
			panic(fmt.Sprintf("Currency conversion error: %v", err1))
		}
		
		// Then convert from USD to target
		result, err = convertCurrency(usdAmount, "USD", toCode)
	}
	
	if err != nil {
		panic(fmt.Sprintf("Currency conversion error: %v", err))
	}
	return result
}

// temperatureConvert handles temperature conversions with proper offset handling
func temperatureConvert(amount *Number, from, to UnitDef) *Number {
	// Handle F -> C conversion (with offset for absolute temperatures)
	if from.name == "°F" && to.name == "°C" {
		if !from.delta && !to.delta {
			// Absolute temperature: F to C = (F - 32) * 5/9
			amount = sub(amount, newNumber(32))
		}
		// Apply scale factor: 5/9
		return mul(amount, newRationalNumber(5, 9))
	}
	
	// Handle C -> F conversion (with offset for absolute temperatures)  
	if from.name == "°C" && to.name == "°F" {
		// Apply scale factor: 9/5
		result := mul(amount, newRationalNumber(9, 5))
		if !from.delta && !to.delta {
			// Absolute temperature: C to F = C * 9/5 + 32
			result = add(result, newNumber(32))
		}
		return result
	}
	
	// Delta to absolute conversion for addition operations
	if from.delta && !to.delta {
		// Delta temperature can be added to absolute temperature
		// Convert delta scale if needed: dF -> C, dC -> F
		if from.name == "°FΔ" && to.name == "°C" {
			return mul(amount, newRationalNumber(5, 9))
		}
		if from.name == "°CΔ" && to.name == "°F" {
			return mul(amount, newRationalNumber(9, 5))
		}
		// Same scale: dC -> C, dF -> F (no conversion needed)
		if (from.name == "°CΔ" && to.name == "°C") || (from.name == "°FΔ" && to.name == "°F") {
			return amount
		}
	}
	
	// Delta to delta conversion
	if from.delta && to.delta {
		if from.name == "°FΔ" && to.name == "°CΔ" {
			return mul(amount, newRationalNumber(5, 9))
		}
		if from.name == "°CΔ" && to.name == "°FΔ" {
			return mul(amount, newRationalNumber(9, 5))
		}
		// Same delta units
		if from.name == to.name {
			return amount
		}
	}
	
	panic(fmt.Sprintf("Unsupported temperature conversion: %s -> %s", from.name, to.name))
}

// conversion factors are exact rational numbers to preserve precision
var UNITS = map[string]UnitDef{
	"nm": {name: "nm", description: "nanometers", dimension: Length, factor: newRationalNumber(1, 1_000_000_000)},
	"um": {name: "um", description: "micrometers", dimension: Length, factor: newRationalNumber(1, 1_000_000)},
	"mm": {name: "mm", description: "millimeters", dimension: Length, factor: newRationalNumber(1, 1_000)},
	"cm": {name: "cm", description: "centimeters", dimension: Length, factor: newRationalNumber(1, 100)},
	"m":  {name: "m", description: "meters", dimension: Length, factor: newNumber(1)},
	"km": {name: "km", description: "kilometers", dimension: Length, factor: newNumber(1000)},

	"in": {name: "in", description: "inches", dimension: Length, factor: newRationalNumber(254, 10000)},   // 0.0254 by definition
	"ft": {name: "ft", description: "feet", dimension: Length, factor: newRationalNumber(3048, 10000)},    // 0.0254 * 12
	"yd": {name: "yd", description: "yards", dimension: Length, factor: newRationalNumber(9144, 10000)},   // 0.0254 * 36
	"mi": {name: "mi", description: "miles", dimension: Length, factor: newRationalNumber(1609344, 1000)}, // 0.0254 * 12 * 5280

	"ug": {name: "ug", description: "micrograms", dimension: Mass, factor: newRationalNumber(1, 1_000_000)},
	"mg": {name: "mg", description: "milligrams", dimension: Mass, factor: newRationalNumber(1, 1_000)},
	"g":  {name: "g", description: "grams", dimension: Mass, factor: newNumber(1)},
	"kg": {name: "kg", description: "kilograms", dimension: Mass, factor: newNumber(1000)},
	"oz": {name: "oz", description: "ounces", dimension: Mass, factor: newRationalNumber(45359237, 1600000)}, // 453.59237 / 16
	"lb": {name: "lb", description: "pounds", dimension: Mass, factor: newRationalNumber(45359237, 100000)},  // 453.59237 by definition

	"ml": {name: "ml", description: "milliliters", dimension: Volume, factor: newRationalNumber(1, 1000)},
	"cl": {name: "cl", description: "centiliters", dimension: Volume, factor: newRationalNumber(1, 100)},
	"dl": {name: "dl", description: "deciliters", dimension: Volume, factor: newRationalNumber(1, 10)},
	"l":  {name: "l", description: "liters", dimension: Volume, factor: newNumber(1)},

	"foz": {name: "foz", description: "fl. ounces", dimension: Volume, factor: newRationalNumber(3785411784, 128000000000)}, // 3.785411784 / 128
	"cup": {name: "cup", description: "cups", dimension: Volume, factor: newRationalNumber(473176473, 2000000000)},          // 3.785411784 / 16
	"pt":  {name: "pt", description: "pints", dimension: Volume, factor: newRationalNumber(473176473, 1000000000)},          // 3.785411784 / 8
	"qt":  {name: "qt", description: "quarts", dimension: Volume, factor: newRationalNumber(946352946, 1000000000)},         // 3.785411784 / 4
	"gal": {name: "gal", description: "us gallons", dimension: Volume, factor: newRationalNumber(3785411784, 1000000000)},   // 231 cubic inches by definition

	"C":  {name: "°C", description: "celsius", dimension: Temperature, factorFunction: temperatureConvert},
	"°C": {name: "°C", description: "celsius", dimension: Temperature, factorFunction: temperatureConvert},
	"F":  {name: "°F", description: "farenheit", dimension: Temperature, factorFunction: temperatureConvert},
	"°F": {name: "°F", description: "farenheit", dimension: Temperature, factorFunction: temperatureConvert},
	"dC": {name: "°CΔ", description: "delta celsius", dimension: Temperature, delta: true, factorFunction: temperatureConvert},
	"dF": {name: "°FΔ", description: "delta farenheit", dimension: Temperature, delta: true, factorFunction: temperatureConvert},

	"s":   {name: "s", description: "seconds", dimension: Time, factor: newNumber(1)},
	"min": {name: "min", description: "minutes", dimension: Time, factor: newNumber(60)},
	"hr":  {name: "hr", description: "hours", dimension: Time, factor: newNumber(3600)},

	// Currency units - USD is base (uses factor), others use dynamic conversion
	"usd": {name: "usd", description: "us dollars", dimension: Currency, factor: newNumber(1)},
	"$":   {name: "$", description: "us dollars", dimension: Currency, factor: newNumber(1)},
	"eur": {name: "eur", description: "euros", dimension: Currency, factorFunction: currencyConvert},
	"€":   {name: "€", description: "euros", dimension: Currency, factorFunction: currencyConvert},
	"gbp": {name: "gbp", description: "british pounds", dimension: Currency, factorFunction: currencyConvert},
	"£":   {name: "£", description: "british pounds", dimension: Currency, factorFunction: currencyConvert},
	"yen": {name: "yen", description: "japanese yen", dimension: Currency, factorFunction: currencyConvert},
	"jpy": {name: "jpy", description: "japanese yen", dimension: Currency, factorFunction: currencyConvert},
	"¥":   {name: "¥", description: "japanese yen", dimension: Currency, factorFunction: currencyConvert},
	"btc": {name: "btc", description: "bitcoin", dimension: Currency, factorFunction: currencyConvert},
}

// 2 sets of units are compatible if they are of the same power in all dimensions
func (u *Units) compatible(other Units) bool {
	result := true

	for i := range u {
		if u[i].power != other[i].power {
			result = false
			break
		}
	}

	return result
}

// temperatureAdditionValid checks if two temperature units can be added
func temperatureAdditionValid(left, right Units) bool {
	leftTemp := left[Temperature]
	rightTemp := right[Temperature]
	
	// If neither has temperature units, not applicable
	if leftTemp.power == 0 && rightTemp.power == 0 {
		return true
	}
	
	// Both must be power 1 for addition
	if leftTemp.power != 1 || rightTemp.power != 1 {
		return false
	}
	
	// Valid combinations:
	// 1. Same absolute units: C + C, F + F
	if leftTemp.name == rightTemp.name {
		return true
	}
	
	// 2. Delta + Absolute: dC + C, dC + F, dF + C, dF + F
	if leftTemp.delta || rightTemp.delta {
		return true
	}
	
	// 3. Different absolute units: C + F (INVALID)
	return false
}

func (u *Units) empty() bool {
	result := true

	for _, unit := range u {
		if unit.power != 0 {
			result = false
			break
		}
	}

	return result
}

func unitUnaryOp(op string, left Value) Value {
	switch op {
	case "r":
		for i := range left.units {
			left.units[i].power *= -1
		}
	default:
		panic(fmt.Sprintf("Unimplmented units unary op: '%s'", op))
	}

	return left
}

func (v Value) MulUnit(other Value) {
	for i := range v.units {
		if v.units[i].power == 0 {
			v.units[i] = other.units[i]
		} else {
			v.units[i].power += other.units[i].power
		}
	}
}

func unitBinaryOp(op string, left, right Value) Value {
	switch op {
	case "*", ".", DOT:
		for i := range left.units {
			if left.units[i].power == 0 {
				left.units[i] = right.units[i]
			} else {
				left.units[i].power += right.units[i].power
			}
		}
	case "**", "pow":
		// TODO: need to handle 1/2, 1/3, 1/4 , etc
		var exponent int = -1
		var integral bool
		if right.number.Rat.IsInt() {
			exponent = int(right.number.Rat.Num().Int64())
			integral = true
		}
		for i := range left.units {
			if left.units[i].power == 0 || exponent == 0 {
				left.units[i] = right.units[i]
			} else if exponent > 0 {
				if !integral {
					die("Can only raise dimensions to integral powers, got %v", right.number)
				}
				left.units[i].power *= exponent
			} else {
				if !integral {
					die("Can only raise dimensions to integral powers, got %v", right.number)
				}
				left.units[i].power /= exponent
			}
		}
	case "/":
		for i := range left.units {
			if left.units[i].power == 0 {
				left.units[i] = right.units[i]
				left.units[i].power = -left.units[i].power
			} else {
				left.units[i].power -= right.units[i].power
			}
		}
	default:
		panic(fmt.Sprintf("Unimplmented units binary op: '%s'", op))
	}

	return left
}

func parseUnits(input string) (Units, bool) {
	var units Units

	if input == "num" { // remove units
		return units, true
	}

	sepRe := regexp.MustCompile(`(^[.*·/])`)
	re := regexp.MustCompile(`^([°a-zA-Z$€£¥]+)(\^(\d+))?`)
	nextPosition := 0
	factor := 1
	if rune(input[0]) == '/' && len(input) > 1 { // no numerator
		nextPosition = 1
		factor = -1
	}

	for {
		match := re.FindStringSubmatch(input[nextPosition:])
		if match == nil {
			break
		}

		var power int = 1
		if match[3] != "" {
			var err error
			power, err = strconv.Atoi(match[3])
			if err != nil {
				break
			}
		}

		if unitDef, ok := UNITS[match[1]]; ok {
			units[unitDef.dimension] = Unit{unitDef, units[unitDef.dimension].power + factor*power}
		} else {
			return units, false
		}

		nextPosition += len(match[0])
		if nextPosition >= len(input) { // end of input
			break
		}

		sepMatch := sepRe.FindStringSubmatch(input[nextPosition:])
		if sepMatch == nil {
			break // unexpected char
		} else {
			if sepMatch[1] == "/" {
				if factor == 1 {
					factor = -1
				} else {
					break // second instance of /
				}
			}
			nextPosition += len(sepMatch[1])
		}
	}

	if nextPosition == len(input) { // reached end of input
		return units, true
	} else {
		return units, false
	}
}

func (v Units) String() string {
	var parts []string
	denominator := false
	for _, unit := range v {
		if unit.power > 0 {
			parts = append(parts, unit.String())
		} else if unit.power < 0 {
			denominator = true
		}
	}
	result := strings.Join(parts, DOT)
	if denominator {
		parts = parts[:0] // clear the parts
		for _, unit := range v {
			if unit.power < 0 {
				parts = append(parts, unit.String())
			}
		}
		result += "/" + strings.Join(parts, DOT)
	}

	return result
}

// should be used from Units.String; stringifies with absolute value of power
func (u Unit) String() string {
	absPower := u.power
	if u.power < 0 {
		absPower = -u.power

	}
	if absPower == 1 {
		return u.name
	}
	return fmt.Sprintf("%s^%d", u.name, absPower)
}

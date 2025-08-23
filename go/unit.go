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

// Dimension represents the type of unit (Length, Time, Mass, Temperature, etc).
type Dimension int

const (
	Mass Dimension = iota
	Length
	Time
	Volume
	Temperature
	Currency
	Current
	NumDimension
)

type BaseUnit struct {
	name           string
	description    string
	dimension      Dimension
	factor         *Number                                   // for simple scaling, nil for dynamic conversion
	delta          bool                                      // only applicable to Temperature
	factorFunction func(*Number, BaseUnit, BaseUnit) *Number // dynamic conversion function
}

type UnitPower struct {
	BaseUnit
	power int
}

type Unit [NumDimension]UnitPower

// conversion factors are exact rational numbers to preserve precision
var UNITS = map[string]Unit{
	// length
	"m": {
		Length: UnitPower{BaseUnit{name: "m", description: "meters", dimension: Length, factor: newNumber(1)}, 1},
	},

	"in": {
		Length: UnitPower{BaseUnit{name: "in", description: "inches", dimension: Length, factor: newRationalNumber(254, 10_000)}, 1},
	},
	"ft": {
		Length: UnitPower{BaseUnit{name: "ft", description: "feet", dimension: Length, factor: newRationalNumber(254*12, 10_000)}, 1},
	},
	"yd": {
		Length: UnitPower{BaseUnit{name: "yd", description: "yards", dimension: Length, factor: newRationalNumber(254*36, 10_000)}, 1},
	},
	"mi": {
		Length: UnitPower{BaseUnit{name: "mi", description: "miles", dimension: Length, factor: newRationalNumber(254*12*5280, 10_000)}, 1},
	},

	// mass
	"g": {
		Mass: UnitPower{BaseUnit{name: "g", description: "grams", dimension: Mass, factor: newNumber(1)}, 1},
	},

	"oz": {
		Mass: UnitPower{BaseUnit{name: "oz", description: "ounces", dimension: Mass, factor: newRationalNumber(45359237, 16*100_000)}, 1},
	},
	"lb": {
		Mass: UnitPower{BaseUnit{name: "lb", description: "pounds", dimension: Mass, factor: newRationalNumber(45359237, 100_000)}, 1},
	},

	// volume
	"l": {
		Volume: UnitPower{BaseUnit{name: "l", description: "liters", dimension: Volume, factor: newNumber(1)}, 1},
	},

	"foz": {
		Volume: UnitPower{BaseUnit{name: "foz", description: "fl. ounces", dimension: Volume, factor: newRationalNumber(3785411784, 128*1_000_000_000)}, 1},
	},
	"cup": {
		Volume: UnitPower{BaseUnit{name: "cup", description: "cups", dimension: Volume, factor: newRationalNumber(3785411784, 16*1_000_000_000)}, 1},
	},
	"pt": {
		Volume: UnitPower{BaseUnit{name: "pt", description: "pints", dimension: Volume, factor: newRationalNumber(3785411784, 8*1_000_000_000)}, 1},
	},
	"qt": {
		Volume: UnitPower{BaseUnit{name: "qt", description: "quarts", dimension: Volume, factor: newRationalNumber(3785411784, 4*1_000_000_000)}, 1},
	},
	"gal": {
		Volume: UnitPower{BaseUnit{name: "gal", description: "us gallons", dimension: Volume, factor: newRationalNumber(3785411784, 1_000_000_000)}, 1},
	},

	// temperature
	"C": {
		Temperature: UnitPower{BaseUnit{name: "°C", description: "celsius", dimension: Temperature, factorFunction: temperatureConvert}, 1},
	},
	"°C": {
		Temperature: UnitPower{BaseUnit{name: "°C", description: "celsius", dimension: Temperature, factorFunction: temperatureConvert}, 1},
	},
	"F": {
		Temperature: UnitPower{BaseUnit{name: "°F", description: "farenheit", dimension: Temperature, factorFunction: temperatureConvert}, 1},
	},
	"°F": {
		Temperature: UnitPower{BaseUnit{name: "°F", description: "farenheit", dimension: Temperature, factorFunction: temperatureConvert}, 1},
	},
	"dC": {
		Temperature: UnitPower{BaseUnit{name: "°CΔ", description: "delta celsius", dimension: Temperature, delta: true, factorFunction: temperatureConvert}, 1},
	},
	"dF": {
		Temperature: UnitPower{BaseUnit{name: "°FΔ", description: "delta farenheit", dimension: Temperature, delta: true, factorFunction: temperatureConvert}, 1},
	},

	// time
	"s": {
		Time: UnitPower{BaseUnit{name: "s", description: "seconds", dimension: Time, factor: newNumber(1)}, 1},
	},
	"min": {
		Time: UnitPower{BaseUnit{name: "min", description: "minutes", dimension: Time, factor: newNumber(60)}, 1},
	},
	"hr": {
		Time: UnitPower{BaseUnit{name: "hr", description: "hours", dimension: Time, factor: newNumber(3600)}, 1},
	},

	// current
	"A": {
		Current: UnitPower{BaseUnit{name: "A", description: "amperes", dimension: Current, factor: newNumber(1)}, 1},
	},

	// currency - USD is base (uses factor), others use dynamic conversion
	"usd": {
		Currency: UnitPower{BaseUnit{name: "usd", description: "us dollars", dimension: Currency, factor: newNumber(1)}, 1},
	},
	"$": {
		Currency: UnitPower{BaseUnit{name: "$", description: "us dollars", dimension: Currency, factor: newNumber(1)}, 1},
	},
	"eur": {
		Currency: UnitPower{BaseUnit{name: "eur", description: "euros", dimension: Currency, factorFunction: currencyConvert}, 1},
	},
	"€": {
		Currency: UnitPower{BaseUnit{name: "€", description: "euros", dimension: Currency, factorFunction: currencyConvert}, 1},
	},
	"gbp": {
		Currency: UnitPower{BaseUnit{name: "gbp", description: "british pounds", dimension: Currency, factorFunction: currencyConvert}, 1},
	},
	"£": {
		Currency: UnitPower{BaseUnit{name: "£", description: "british pounds", dimension: Currency, factorFunction: currencyConvert}, 1},
	},
	"yen": {
		Currency: UnitPower{BaseUnit{name: "yen", description: "japanese yen", dimension: Currency, factorFunction: currencyConvert}, 1},
	},
	"jpy": {
		Currency: UnitPower{BaseUnit{name: "jpy", description: "japanese yen", dimension: Currency, factorFunction: currencyConvert}, 1},
	},
	"¥": {
		Currency: UnitPower{BaseUnit{name: "¥", description: "japanese yen", dimension: Currency, factorFunction: currencyConvert}, 1},
	},
	"btc": {
		Currency: UnitPower{BaseUnit{name: "btc", description: "bitcoin", dimension: Currency, factorFunction: currencyConvert}, 1},
	},

	// derived units
	// joules J = kg⋅m²⋅s⁻²
	"J": {
		Mass:   UnitPower{BaseUnit{name: "kg", dimension: Mass, factor: newNumber(1_000)}, 1},
		Length: UnitPower{BaseUnit{name: "m", dimension: Length, factor: newNumber(1)}, 2},
		Time:   UnitPower{BaseUnit{name: "s", dimension: Time, factor: newNumber(1)}, -2},
	},
	// newtons N = kg⋅m⋅s⁻²
	"N": {
		Mass:   UnitPower{BaseUnit{name: "kg", dimension: Mass, factor: newNumber(1_000)}, 1},
		Length: UnitPower{BaseUnit{name: "m", dimension: Length, factor: newNumber(1)}, 1},
		Time:   UnitPower{BaseUnit{name: "s", dimension: Time, factor: newNumber(1)}, -2},
	},
	// volts V = kg⋅m²⋅s⁻³⋅A⁻¹
	"V": {
		Mass:    UnitPower{BaseUnit{name: "kg", dimension: Mass, factor: newNumber(1_000)}, 1},
		Length:  UnitPower{BaseUnit{name: "m", dimension: Length, factor: newNumber(1)}, 2},
		Time:    UnitPower{BaseUnit{name: "s", dimension: Time, factor: newNumber(1)}, -3},
		Current: UnitPower{BaseUnit{name: "A", dimension: Current, factor: newNumber(1)}, -1},
	},
	// watts W = kg⋅m²⋅s⁻³
	"W": {
		Mass:   UnitPower{BaseUnit{name: "kg", dimension: Mass, factor: newNumber(1_000)}, 1},
		Length: UnitPower{BaseUnit{name: "m", dimension: Length, factor: newNumber(1)}, 2},
		Time:   UnitPower{BaseUnit{name: "s", dimension: Time, factor: newNumber(1)}, -3},
	},
	// ohms Ω = kg⋅m²⋅s⁻³⋅A⁻²
	"Ω": {
		Mass:    UnitPower{BaseUnit{name: "kg", dimension: Mass, factor: newNumber(1_000)}, 1},
		Length:  UnitPower{BaseUnit{name: "m", dimension: Length, factor: newNumber(1)}, 2},
		Time:    UnitPower{BaseUnit{name: "s", dimension: Time, factor: newNumber(1)}, -3},
		Current: UnitPower{BaseUnit{name: "A", dimension: Current, factor: newNumber(1)}, -2},
	},
}

// SI Prefix definitions with power of 10
type SIPrefix struct {
	symbol string
	name   string
	power  int // power of 10
}

var SI_PREFIXES = []SIPrefix{
	{"da", "deca", 1},
	{"h", "hecto", 2},
	{"k", "kilo", 3},
	{"M", "mega", 6},
	{"G", "giga", 9},
	{"T", "tera", 12},
	{"P", "peta", 15},
	{"E", "exa", 18},

	{"d", "deci", -1},
	{"c", "centi", -2},
	{"m", "milli", -3},
	{"μ", "micro", -6},
	{"u", "micro", -6},
	{"n", "nano", -9},
	{"p", "pico", -12},
	{"f", "femto", -15},
	{"a", "atto", -18},
}

// currencyConvert handles any currency conversion, including multi-currency via USD
func currencyConvert(amount *Number, from, to BaseUnit) *Number {
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
func temperatureConvert(amount *Number, from, to BaseUnit) *Number {
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

// Units that accept SI prefixes
var UNITS_FOR_PREFIXES = []string{"m", "g", "l", "A", "V", "W", "Ω"}

func generatePrefixedUnits() {
	for _, baseUnitName := range UNITS_FOR_PREFIXES {
		if baseUnit, exists := UNITS[baseUnitName]; exists {
			for _, prefix := range SI_PREFIXES {
				prefixedSymbol := prefix.symbol + baseUnitName

				if _, exists := UNITS[prefixedSymbol]; exists {
					panic(fmt.Sprintf("Unit conflict, attempt to redefine '%s'", prefixedSymbol))
				}

				// Make a copy of the entire Unit structure
				var newUnit Unit
				copy(newUnit[:], baseUnit[:])

				// Find the first non-zero power base unit and apply prefix factor
				prefixFactor := pow(newNumber(10), newNumber(prefix.power))
				for dim, unit := range newUnit {
					if unit.power != 0 {
						// Apply prefix factor to this unit's factor
						if unit.factor != nil {
							newUnit[dim].factor = mul(unit.factor, prefixFactor)
						} else {
							newUnit[dim].factor = prefixFactor
						}
						// Update the name to include prefix
						newUnit[dim].name = prefixedSymbol
						newUnit[dim].description = prefix.name + unit.description
						break // Only modify the first non-zero power unit
					}
				}

				UNITS[prefixedSymbol] = newUnit
			}
		}
	}

	// Add word aliases for derived units (TODO: these don't support SI prefixes yet)
	UNITS["ohm"] = UNITS["Ω"]
}

var DERIVED_UNIT_NAMES = []string{"J", "N", "Ω", "V", "W"}

// 2 sets of units are compatible if they are of the same power in all dimensions
func (u *Unit) compatible(other Unit) bool {
	result := true

	for i := range u {
		if u[i].power != other[i].power {
			result = false
			break
		}
	}

	if false {
		if options.debug {
			fmt.Printf("Comparing units: %v\n", result)
			for i := range u {
				fmt.Printf("  %4s  %3d  %3d\n", u[i].name, u[i].power, other[i].power)
			}
		}
	}

	return result
}

// temperatureAdditionValid checks if two temperature units can be added
func temperatureAdditionValid(left, right Unit) bool {
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

// checks if temperature multiplication is allowed
func temperatureMultiplicationValid(left, right Unit) bool {
	// As long as one side does not have temperature units, multiplication is allowed (e.g., 2 * 20°C)
	return left[Temperature].power == 0 || right[Temperature].power == 0
}

func (u *Unit) empty() bool {
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

// fromSuperscript converts superscript Unicode to regular numbers
func fromSuperscript(s string) string {
	superscriptMap := map[rune]rune{
		'⁰': '0', '¹': '1', '²': '2', '³': '3', '⁴': '4',
		'⁵': '5', '⁶': '6', '⁷': '7', '⁸': '8', '⁹': '9',
		'⁻': '-',
	}

	result := ""
	for _, r := range s {
		if normal, exists := superscriptMap[r]; exists {
			result += string(normal)
		} else {
			result += string(r)
		}
	}
	return result
}

func parseUnits(input string) (Unit, bool) {
	var units Unit

	if input == "num" { // remove units
		return units, true
	}

	sepRe := regexp.MustCompile(`(^[.*·/])`)
	// Updated regex to handle superscripts and negative powers
	re := regexp.MustCompile(`^([°a-zA-Z$€£¥Ωμ]+)(\^(-?\d+)|([⁰¹²³⁴⁵⁶⁷⁸⁹⁻]+))?`)
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
		var err error

		if match[3] != "" {
			// Handle ^-digit or ^digit format
			power, err = strconv.Atoi(match[3])
			if err != nil {
				break
			}
		} else if match[4] != "" {
			// Handle superscript format
			normalizedPower := fromSuperscript(match[4])
			power, err = strconv.Atoi(normalizedPower)
			if err != nil {
				break
			}
		}

		unitName := match[1]

		// Handle units - all units (base and derived) are in UNITS table
		if unitUnit, ok := UNITS[unitName]; ok {
			// Handle regular units - add all dimensions from the Unit array
			for dim, unit := range unitUnit {
				if unit.power != 0 {
					units[dim] = UnitPower{unit.BaseUnit, units[dim].power + factor*power*unit.power}
				}
			}
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

func (v Unit) Name() string {
	name := v.String()
	if name == "" {
		name = "dimensionless"
	}

	return name
}

func (v Unit) String() string {
	// Skip derived unit matching if --base option is enabled
	if !options.base {
		// Try to match with base derived units only - use DERIVED_UNIT_NAMES
		for _, symbol := range DERIVED_UNIT_NAMES {
			if derivedUnit, exists := UNITS[symbol]; exists {
				if unitsMatch(v, derivedUnit) {
					return symbol
				}
			}
		}
	}

	// Use base units only (or if no derived unit matches)
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

// unitsMatch checks if two Unit are equivalent
func unitsMatch(units1, units2 Unit) bool {
	for i := 0; i < len(units1); i++ {
		if units1[i].power != units2[i].power {
			return false
		}
	}
	return true
}

// toSuperscript converts a number to its superscript Unicode representation
func toSuperscript(n int) string {
	if n == 0 {
		return "⁰"
	}
	if n < 0 {
		return "⁻" + toSuperscript(-n)
	}

	superscripts := map[rune]rune{
		'0': '⁰', '1': '¹', '2': '²', '3': '³', '4': '⁴',
		'5': '⁵', '6': '⁶', '7': '⁷', '8': '⁸', '9': '⁹',
	}

	numStr := fmt.Sprintf("%d", n)
	result := ""
	for _, digit := range numStr {
		if superscript, exists := superscripts[digit]; exists {
			result += string(superscript)
		}
	}
	return result
}

// should be used from Unit.String; stringifies with absolute value of power
func (u UnitPower) String() string {
	absPower := u.power
	if u.power < 0 {
		absPower = -u.power
	}
	if absPower == 1 {
		return u.name
	}

	// Use superscript by default, unless -S option is specified
	if options.superscript {
		return u.name + toSuperscript(absPower)
	} else {
		return fmt.Sprintf("%s^%d", u.name, absPower)
	}
}

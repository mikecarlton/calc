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
	Current
	ElectricalVolt // Special dimension for voltage units
	ElectricalWatt // Special dimension for power units
	ElectricalOhm  // Special dimension for resistance units
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

// DerivedUnit represents a unit that can be expressed in terms of base units
type DerivedUnit struct {
	name        string
	symbol      string
	description string
	baseUnits   Units // The combination of base units this represents
}

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

// electricalConvert handles derived electrical unit conversions
func electricalConvert(amount *Number, from, to UnitDef) *Number {
	// This should not be called directly - derived units should be converted
	// to base units during parsing and back to derived during display
	return amount
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

// Base units that accept SI prefixes
var BASE_UNITS_FOR_PREFIXES = []string{"m", "g", "l", "A"}

// Derived units that accept SI prefixes
var DERIVED_UNITS_FOR_PREFIXES = []string{"V", "W", "Ω"}

// Electrical units use regular dimensions but with special handling
// We'll treat volts, watts, and ohms as regular units that can convert between prefixes

func generatePrefixedUnits() {
	if options.debug {
		fmt.Printf("Generating SI prefixed units\n")
	}

	for _, baseUnit := range BASE_UNITS_FOR_PREFIXES {
		if baseUnitDef, exists := UNITS[baseUnit]; exists {
			for _, prefix := range SI_PREFIXES {
				prefixedSymbol := prefix.symbol + baseUnit

				if _, exists := UNITS[prefixedSymbol]; exists {
					panic(fmt.Sprintf("Unit conflict, attempt to redefine '%s'", prefixedSymbol))
				}

				if options.debug {
					fmt.Printf("  %s (factor=10^%d, desc=%s%s)\n",
						prefixedSymbol, prefix.power, prefix.name, baseUnitDef.description)
				}

				// Calculate factor: base_factor * 10^prefix_power
				prefixFactor := pow(newNumber(10), newNumber(prefix.power))
				factor := mul(baseUnitDef.factor, prefixFactor)

				UNITS[prefixedSymbol] = UnitDef{
					name:        prefixedSymbol,
					description: prefix.name + baseUnitDef.description,
					dimension:   baseUnitDef.dimension,
					factor:      factor,
				}
			}
		}
	}

	// Generate prefixed derived units as regular units with their own dimensions
	derivedDimensionMap := map[string]Dimension{
		"V": ElectricalVolt,
		"W": ElectricalWatt,
		"Ω": ElectricalOhm,
	}

	for _, derivedUnit := range DERIVED_UNITS_FOR_PREFIXES {
		if dimension, exists := derivedDimensionMap[derivedUnit]; exists {
			// Add base derived unit as regular unit
			UNITS[derivedUnit] = UnitDef{
				name:        derivedUnit,
				description: derivedUnit,
				dimension:   dimension,
				factor:      newNumber(1),
			}

			if options.debug {
				fmt.Printf("  Generated: %s (factor=1, desc=%s)\n", derivedUnit, derivedUnit)
			}

			// Generate prefixed versions
			for _, prefix := range SI_PREFIXES {
				prefixedSymbol := prefix.symbol + derivedUnit

				// Skip if conflicts
				if _, exists := UNITS[prefixedSymbol]; exists {
					continue
				}

				// Calculate prefix factor: 10^prefix_power
				prefixFactor := pow(newNumber(10), newNumber(prefix.power))

				UNITS[prefixedSymbol] = UnitDef{
					name:        prefixedSymbol,
					description: prefix.name + derivedUnit,
					dimension:   dimension,
					factor:      prefixFactor,
				}
			}
		}
	}

	// Add word aliases for derived units
	UNITS["volt"] = UnitDef{name: "V", description: "volt", dimension: ElectricalVolt, factor: newNumber(1)}
	UNITS["watt"] = UnitDef{name: "W", description: "watt", dimension: ElectricalWatt, factor: newNumber(1)}
	UNITS["ohm"] = UnitDef{name: "Ω", description: "ohm", dimension: ElectricalOhm, factor: newNumber(1)}

	// Don't add prefixed derived units to DERIVED_UNITS table
	// This prevents display confusion - only base derived units (V, W, Ω) should be in DERIVED_UNITS
}

// Table of derived units that can be factored from base units
var DERIVED_UNITS = map[string]DerivedUnit{
	"V": {
		name:        "V",
		symbol:      "V",
		description: "volt",
		baseUnits: Units{
			Mass:    Unit{UnitDef{name: "kg", dimension: Mass}, 1},    // kg
			Length:  Unit{UnitDef{name: "m", dimension: Length}, 2},   // m²
			Time:    Unit{UnitDef{name: "s", dimension: Time}, -3},    // s⁻³
			Current: Unit{UnitDef{name: "A", dimension: Current}, -1}, // A⁻¹
		},
	},
	"W": {
		name:        "W",
		symbol:      "W",
		description: "watt",
		baseUnits: Units{
			Mass:   Unit{UnitDef{name: "kg", dimension: Mass}, 1},  // kg
			Length: Unit{UnitDef{name: "m", dimension: Length}, 2}, // m²
			Time:   Unit{UnitDef{name: "s", dimension: Time}, -3},  // s⁻³
		},
	},
	"Ω": {
		name:        "Ω",
		symbol:      "Ω",
		description: "ohm",
		baseUnits: Units{
			Mass:    Unit{UnitDef{name: "kg", dimension: Mass}, 1},    // kg
			Length:  Unit{UnitDef{name: "m", dimension: Length}, 2},   // m²
			Time:    Unit{UnitDef{name: "s", dimension: Time}, -3},    // s⁻³
			Current: Unit{UnitDef{name: "A", dimension: Current}, -2}, // A⁻²
		},
	},
}

// conversion factors are exact rational numbers to preserve precision
var UNITS = map[string]UnitDef{
	// length
	"m": {name: "m", description: "meters", dimension: Length, factor: newNumber(1)},

	"in": {name: "in", description: "inches", dimension: Length, factor: newRationalNumber(254, 10_000)}, // 0.0254 by definition
	"ft": {name: "ft", description: "feet", dimension: Length, factor: newRationalNumber(254*12, 10_000)},
	"yd": {name: "yd", description: "yards", dimension: Length, factor: newRationalNumber(254*36, 10_000)},
	"mi": {name: "mi", description: "miles", dimension: Length, factor: newRationalNumber(254*12*5280, 10_000)},

	// mass
	"g": {name: "g", description: "grams", dimension: Mass, factor: newNumber(1)},

	"oz": {name: "oz", description: "ounces", dimension: Mass, factor: newRationalNumber(45359237, 16*100_000)},
	"lb": {name: "lb", description: "pounds", dimension: Mass, factor: newRationalNumber(45359237, 100_000)}, // 453.59237 by definition

	// volume
	"l": {name: "l", description: "liters", dimension: Volume, factor: newNumber(1)},

	"foz": {name: "foz", description: "fl. ounces", dimension: Volume, factor: newRationalNumber(3785411784, 128*1_000_000_000)},
	"cup": {name: "cup", description: "cups", dimension: Volume, factor: newRationalNumber(3785411784, 16*1_000_000_000)},
	"pt":  {name: "pt", description: "pints", dimension: Volume, factor: newRationalNumber(3785411784, 8*1_000_000_000)},
	"qt":  {name: "qt", description: "quarts", dimension: Volume, factor: newRationalNumber(3785411784, 4*1_000_000_000)},
	"gal": {name: "gal", description: "us gallons", dimension: Volume, factor: newRationalNumber(3785411784, 1_000_000_000)}, // 231 cubic inches by definition

	// temperature
	"C":  {name: "°C", description: "celsius", dimension: Temperature, factorFunction: temperatureConvert},
	"°C": {name: "°C", description: "celsius", dimension: Temperature, factorFunction: temperatureConvert},
	"F":  {name: "°F", description: "farenheit", dimension: Temperature, factorFunction: temperatureConvert},
	"°F": {name: "°F", description: "farenheit", dimension: Temperature, factorFunction: temperatureConvert},
	"dC": {name: "°CΔ", description: "delta celsius", dimension: Temperature, delta: true, factorFunction: temperatureConvert},
	"dF": {name: "°FΔ", description: "delta farenheit", dimension: Temperature, delta: true, factorFunction: temperatureConvert},

	"s":   {name: "s", description: "seconds", dimension: Time, factor: newNumber(1)},
	"min": {name: "min", description: "minutes", dimension: Time, factor: newNumber(60)},
	"hr":  {name: "hr", description: "hours", dimension: Time, factor: newNumber(3600)},

	// current
	"A":      {name: "A", description: "amperes", dimension: Current, factor: newNumber(1)},
	"ampere": {name: "A", description: "amperes", dimension: Current, factor: newNumber(1)},
	"amp":    {name: "A", description: "amperes", dimension: Current, factor: newNumber(1)},

	// currency - USD is base (uses factor), others use dynamic conversion
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

	if options.debug {
		fmt.Printf("Comparing units: %v\n", result)
		for i := range u {
			fmt.Printf("  %4s  %3d  %3d\n", u[i].name, u[i].power, other[i].power)
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

// checks if temperature multiplication is allowed
func temperatureMultiplicationValid(left, right Units) bool {
	// As long as one side does not have temperature units, multiplication is allowed (e.g., 2 * 20°C)
	return left[Temperature].power == 0 || right[Temperature].power == 0
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
	re := regexp.MustCompile(`^([°a-zA-Z$€£¥Ωμ]+)(\^(\d+))?`)
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

		unitName := match[1]

		// Check if this is a derived unit that needs to be converted to base units
		if derivedUnit, isDerived := DERIVED_UNITS[unitName]; isDerived {
			// Convert derived unit to base units
			for dim, baseUnit := range derivedUnit.baseUnits {
				if baseUnit.power != 0 {
					units[dim] = Unit{baseUnit.UnitDef, units[dim].power + factor*power*baseUnit.power}
				}
			}
		} else if unitDef, ok := UNITS[unitName]; ok {
			// Handle regular units
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

func (v Units) Name() string {
	name := v.String()
	if name == "" {
		name = "dimensionless"
	}

	return name
}

func (v Units) String() string {
	// Try to match with base derived units only (V, W, Ω) not prefixed ones
	baseDerivedUnits := map[string]DerivedUnit{
		"V": DERIVED_UNITS["V"],
		"W": DERIVED_UNITS["W"],
		"Ω": DERIVED_UNITS["Ω"],
	}

	for symbol, derivedUnit := range baseDerivedUnits {
		if unitsMatch(v, derivedUnit.baseUnits) {
			return symbol
		}
	}

	// If no derived unit matches, use the original logic
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

// unitsMatch checks if two Units are equivalent
func unitsMatch(units1, units2 Units) bool {
	for i := 0; i < len(units1); i++ {
		if units1[i].power != units2[i].power {
			return false
		}
	}
	return true
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

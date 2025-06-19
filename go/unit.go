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
	name        string
	description string
	dimension   Dimension
	factor      *Number
	delta       bool // only applicable to Temperature
}

type Unit struct {
	UnitDef
	power int
}

type Units [NumDimension]Unit

// TODO: use Rat for factor?? (need to support it in Number) or two factors (mul & div?) ??
var UNITS = map[string]UnitDef{
	"nm": {name: "nm", description: "nanometers", dimension: Length, factor: NewNumber(1.0 / (1000.0 * 1000.0 * 1000.0))},
	"um": {name: "um", description: "micrometers", dimension: Length, factor: NewNumber(1.0 / (1000.0 * 1000.0))},
	"mm": {name: "mm", description: "millimeters", dimension: Length, factor: NewNumber(1.0 / 1000.0)},
	"cm": {name: "cm", description: "centimeters", dimension: Length, factor: NewNumber(1.0 / 100.0)},
	"m":  {name: "m", description: "meters", dimension: Length, factor: NewNumber(1)},
	"km": {name: "km", description: "kilometers", dimension: Length, factor: NewNumber(1000)},

	"in": {name: "in", description: "inches", dimension: Length, factor: NewNumber(0.0254)}, // by definition
	"ft": {name: "ft", description: "feet", dimension: Length, factor: NewNumber(0.0254 * 12.0)},
	"yd": {name: "yd", description: "yards", dimension: Length, factor: NewNumber(0.0254 * 36.0)},
	"mi": {name: "mi", description: "miles", dimension: Length, factor: NewNumber(0.0254 * 12.0 * 5280.0)},

	"ug": {name: "ug", description: "micrograms", dimension: Mass, factor: NewNumber(1.0 / (1000.0 * 1000.0))},
	"mg": {name: "mg", description: "milligrams", dimension: Mass, factor: NewNumber(1.0 / 1000.0)},
	"g":  {name: "g", description: "grams", dimension: Mass, factor: NewNumber(1)},
	"kg": {name: "kg", description: "kilograms", dimension: Mass, factor: NewNumber(1000)},
	"oz": {name: "oz", description: "ounces", dimension: Mass, factor: NewNumber(453.59237 / 16.0)},
	"lb": {name: "lb", description: "pounds", dimension: Mass, factor: NewNumber(453.59237)}, // by definition

	"ml": {name: "ml", description: "milliliters", dimension: Volume, factor: NewNumber(1.0 / 1000.0)},
	"cl": {name: "cl", description: "centiliters", dimension: Volume, factor: NewNumber(1.0 / 100.0)},
	"dl": {name: "dl", description: "deciliters", dimension: Volume, factor: NewNumber(1.0 / 10.0)},
	"l":  {name: "l", description: "liters", dimension: Volume, factor: NewNumber(1)},

	"foz": {name: "foz", description: "fl. ounces", dimension: Volume, factor: NewNumber(3.785411784 / 128.0)},
	"cup": {name: "cup", description: "cups", dimension: Volume, factor: NewNumber(3.785411784 / 16.0)},
	"pt":  {name: "pt", description: "pints", dimension: Volume, factor: NewNumber(3.785411784 / 8.0)},
	"qt":  {name: "qt", description: "quarts", dimension: Volume, factor: NewNumber(3.785411784 / 4.0)},
	"gal": {name: "gal", description: "us gallons", dimension: Volume, factor: NewNumber(3.785411784)}, // 231 cubic inches by definition

	"C":  {name: "°C", description: "celsius", dimension: Temperature, factor: NewNumber(1)},
	"°C": {name: "°C", description: "celsius", dimension: Temperature, factor: NewNumber(1)},
	"F":  {name: "°F", description: "farenheit", dimension: Temperature, factor: NewNumber(5.0 / 9.0)},
	"°F": {name: "°F", description: "farenheit", dimension: Temperature, factor: NewNumber(5.0 / 9.0)},
	"dC": {name: "°CΔ", description: "delta celsius", dimension: Temperature, delta: true, factor: NewNumber(1)},
	"dF": {name: "°FΔ", description: "delta farenheit", dimension: Temperature, delta: true, factor: NewNumber(5.0 / 9.0)},

	"s":   {name: "s", description: "seconds", dimension: Time, factor: NewNumber(1)},
	"min": {name: "min", description: "minutes", dimension: Time, factor: NewNumber(60)},
	"hr":  {name: "hr", description: "hours", dimension: Time, factor: NewNumber(3600)},
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
	re := regexp.MustCompile(`^([°a-zA-Z]+)(\^(\d+))?`)
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

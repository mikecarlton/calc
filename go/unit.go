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
	factor      Number
}

type Unit struct {
	UnitDef
	power int
}

type Units [NumDimension]Unit

// TODO: use Rat for factor?? (need to support it in Number) or two factors (mul & div?) ??
var UNITS = map[string]UnitDef{
	"nm": {name: "nm", description: "nanometers", dimension: Length, factor: newFloat(1.0 / (1000.0 * 1000.0 * 1000.0))},
	"um": {name: "um", description: "micrometers", dimension: Length, factor: newFloat(1.0 / (1000.0 * 1000.0))},
	"mm": {name: "mm", description: "millimeters", dimension: Length, factor: newFloat(1.0 / 1000.0)},
	"cm": {name: "cm", description: "centimeters", dimension: Length, factor: newFloat(1.0 / 100.0)},
	"m":  {name: "m", description: "meters", dimension: Length, factor: newInt(1)},
	"km": {name: "km", description: "kilometers", dimension: Length, factor: newInt(1000)},

	"in": {name: "in", description: "inches", dimension: Length, factor: newFloat(0.0254)},
	"ft": {name: "ft", description: "feet", dimension: Length, factor: newFloat(0.0254 * 12.0)},
	"yd": {name: "yd", description: "yards", dimension: Length, factor: newFloat(0.0254 * 36.0)},
	"mi": {name: "mi", description: "miles", dimension: Length, factor: newFloat(0.0254 * 12.0 * 5280.0)},

	"g":  {name: "g", description: "grams", dimension: Mass, factor: newInt(1)},
	"kg": {name: "kg", description: "kilograms", dimension: Mass, factor: newInt(1000)},
	"oz": {name: "oz", description: "ounces", dimension: Mass, factor: newFloat(28.3495)},
	"lb": {name: "lb", description: "pounds", dimension: Mass, factor: newFloat(28.3495 * 16.0)},

	"ml": {name: "ml", description: "milliliters", dimension: Volume, factor: newFloat(1.0 / 1000.0)},
	"cl": {name: "cl", description: "centiliters", dimension: Volume, factor: newFloat(1.0 / 100.0)},
	"dl": {name: "dl", description: "deciliters", dimension: Volume, factor: newFloat(1.0 / 10.0)},
	"l":  {name: "l", description: "liters", dimension: Volume, factor: newInt(1)},

	"foz": {name: "foz", description: "fl. ounces", dimension: Volume, factor: newFloat(3.78541 / 128.0)},
	"cup": {name: "cup", description: "cups", dimension: Volume, factor: newFloat(3.78541 / 16.0)},
	"pt":  {name: "pt", description: "pints", dimension: Volume, factor: newFloat(3.78541 / 8.0)},
	"qt":  {name: "qt", description: "quarts", dimension: Volume, factor: newFloat(3.78541 / 4.0)},
	"gal": {name: "gal", description: "us gallons", dimension: Volume, factor: newFloat(3.78541)},

	"s":   {name: "s", description: "seconds", dimension: Time, factor: newInt(1)},
	"min": {name: "min", description: "minutes", dimension: Time, factor: newInt(60)},
	"hr":  {name: "hr", description: "hours", dimension: Time, factor: newInt(3600)},
}

func (u *Units) compatible(other Units) bool {
	result := true

	for i, _ := range u {
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

func unitBinaryOp(left, right Units, op string) Units {
	switch op {
	case "*", "•", ".":
		for i := range left {
			if left[i].power == 0 {
				left[i] = right[i]
			} else {
				left[i].power += right[i].power
			}
		}
	case "/":
		for i := range left {
			if left[i].power == 0 {
				left[i] = right[i]
				left[i].power = -left[i].power
			} else {
				left[i].power -= right[i].power
			}
		}
	default:
		panic(fmt.Sprintf("Unimplmented units op: '%s'", op))
	}

	return left
}

func parseUnits(input string) (Units, bool) {
	var units Units

	if input == "num" { // remove units
		return units, true
	}

	re := regexp.MustCompile(`^([a-zA-Z]+)(\^(\d+))?`)
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
		sep := rune(input[nextPosition])
		if sep == '*' || sep == '•' || sep == '.' {
			nextPosition += 1
		} else if sep == '/' {
			if factor == 1 {
				factor = -1
			} else {
				break // second instance of /
			}
			nextPosition += 1
		} else {
			break // unexpected char
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
	result := strings.Join(parts, "•")
	if denominator {
		parts = parts[:0] // clear the parts
		for _, unit := range v {
			if unit.power < 0 {
				parts = append(parts, unit.String())
			}
		}
		result += "/" + strings.Join(parts, "•")
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

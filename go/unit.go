// Copyright 2024 Mike Carlton
// Released under terms of the MIT License:
//   http://www.opensource.org/licenses/mit-license.php

package main

// Dimension represents the type of unit (Length, Time, Mass, Temperature).
type Dimension int

const (
	Mass Dimension = iota
	Length
	Time
	Volume
	Temperature
	Currency
)

type Kind struct {
	name        string
	description string
	dimension   Dimension
	factor      Number
}

type Unit struct {
	kind  Kind
	power int
}

// units is a Array[Dimension] of unit
type Units []Unit

/*
var units = map[string]Kind{
	"mm": Kind{name: "mm", description: "millimeters", dimension: Length, factor: 1.0 / 1000.0},
	"cm": Kind{name: "cm", description: "centimeters", dimension: Length, factor: 1.0 / 100.0},
	"m":  Kind{name: "m", description: "meters", dimension: Length, factor: 1.0},
	"km": Kind{name: "km", description: "kilometers", dimension: Length, factor: 1000.0},

	"in": Kind{name: "in", description: "inches", dimension: Length, factor: 0.0254},
	"ft": Kind{name: "ft", description: "feet", dimension: Length, factor: 0.0254 * 12.0},
	"yd": Kind{name: "yd", description: "yards", dimension: Length, factor: 0.0254 * 36.0},
	"mi": Kind{name: "mi", description: "miles", dimension: Length, factor: 0.0254 * 12.0 * 5280.0},

	"g":  Kind{name: "g", description: "grams", dimension: Mass, factor: 1.0},
	"kg": Kind{name: "kg", description: "kilograms", dimension: Mass, factor: 1000.0},
	"oz": Kind{name: "oz", description: "ounces", dimension: Mass, factor: 28.3495},
	"lb": Kind{name: "lb", description: "pounds", dimension: Mass, factor: 28.3495 * 16.0},

	"ml": Kind{name: "ml", description: "milliliters", dimension: Volume, factor: 1.0 / 1000.0},
	"cl": Kind{name: "cl", description: "centiliters", dimension: Volume, factor: 1.0 / 100.0},
	"dl": Kind{name: "dl", description: "deciliters", dimension: Volume, factor: 1.0 / 10.0},
	"l":  Kind{name: "l", description: "liters", dimension: Volume, factor: 1.0},

	"foz": Kind{name: "foz", description: "fl. ounces", dimension: Volume, factor: 3.78541 / 128.0},
	"cup": Kind{name: "cup", description: "cups", dimension: Volume, factor: 3.78541 / 16.0},
	"pt":  Kind{name: "pt", description: "pints", dimension: Volume, factor: 3.78541 / 8.0},
	"qt":  Kind{name: "qt", description: "quarts", dimension: Volume, factor: 3.78541 / 4.0},
	"gal": Kind{name: "gal", description: "us gallons", dimension: Volume, factor: 3.78541},

	"s":   Kind{name: "s", description: "seconds", dimension: Time, factor: 1.0},
	"min": Kind{name: "min", description: "minutes", dimension: Time, factor: 60.0},
	"hr":  Kind{name: "hr", description: "hours", dimension: Time, factor: 3600.0},
	"day": Kind{name: "day", description: "days", dimension: Time, factor: 86400.0},
}

// need regex: unit1*unit1/unit3*unit4^power •.*
func parseUnits(input string) ([]Unit, bool) {
	if kind, ok := units[input]; ok {
		return []Unit{
			{kind: kind, power: 1},
		}, true
	} else {
		return []Unit{}, false
	}
}

func (v Units) String() string {
	// sort by dimenstion and power > 0 or < 0 (ignore power == 0)
	return v
}

func (v Unit) String() string {
	return v.number.String() + v.units.String()
}
*/

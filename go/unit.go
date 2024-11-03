// Copyright 2024 Mike Carlton
// Released under terms of the MIT License:
//   http://www.opensource.org/licenses/mit-license.php

package main

// UnitType represents the type of unit (Length, Time, Mass, Temperature).
type UnitType int

const (
	Length UnitType = iota
	Time
	Mass
	Temperature
)

type UnitKind struct {
	Name      string
	Dimension UnitType
}

type Unit struct {
	Kind  UnitKind
	Power int
}

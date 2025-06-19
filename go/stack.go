// Copyright 2024 Mike Carlton
// Released under terms of the MIT License:
//   http://www.opensource.org/licenses/mit-license.php

package main

import (
	"fmt"
	"strings"
)

type Stack struct {
	values []Value
}

func newStack() *Stack {
	return &Stack{values: []Value{}}
}

var STACKALIAS = Aliases{
	"dup": "d",
	"pop": "p",
}

var STACKOP = map[string]func(*Stack){
	"x": func(s *Stack) { s.exchange() },
	"d": func(s *Stack) { s.dup() },
	"p": func(s *Stack) {
		if _, err := s.pop(); err != nil {
			die("Stack is empty for '%s', exiting", "pop")
		}
	},
}

func (s *Stack) binaryOp(op string) {
	right, _ := s.pop()
	left, err := s.pop()
	if err != nil {
		die("Not enough arguments for binary operation '%s', exiting", op)
	}

	s.push(left.binaryOp(op, right))
}

func (s *Stack) unaryOp(op string) {
	value, err := s.pop()
	if err != nil {
		die("Not enough arguments for unary operation '%s', exiting", op)
	}

	s.push(value.unaryOp(op))
}

func (s *Stack) apply(units Units) {
	value, err := s.pop()
	if err != nil {
		die("Not enough arguments for '%s', exiting", units)
	}

	s.push(value.apply(units))
}

func (s *Stack) push(v Value) {
	s.values = append(s.values, v)
}

func (s *Stack) pop() (Value, error) {
	if len(s.values) == 0 {
		return Value{}, fmt.Errorf("stack is empty")
	}
	v := s.values[len(s.values)-1]
	s.values = s.values[:len(s.values)-1]

	return v, nil
}

func (s *Stack) peek() (Value, error) {
	if len(s.values) == 0 {
		return Value{}, fmt.Errorf("stack is empty")
	}

	return s.values[len(s.values)-1], nil
}

func (s *Stack) dup() {
	if len(s.values) < 1 {
		die("Stack is empty for '%s', exiting", "duplicate")
	}

	// TODO: need to copy value, otherwise they're aliased
	s.values = append(s.values, s.values[len(s.values)-1])
}

func (s *Stack) exchange() {
	if len(s.values) < 2 {
		die("Not enough arguments for '%s', exiting", "exchange")
	}

	s.values[len(s.values)-1], s.values[len(s.values)-2] = s.values[len(s.values)-2], s.values[len(s.values)-1]
}

func (s *Stack) size() int {
	return len(s.values)
}

func (s *Stack) oneline() string {
	var sb strings.Builder
	separator := ""
	for i, v := range s.values {
		sb.WriteString(fmt.Sprintf("%s%s", separator, v))
		if i == 0 {
			separator = " "
		}
	}
	return sb.String()
}

// return max widths for all enabled base columns
func maxWidths(values []Value) map[int]int {
	widths := make(map[int]int)
	bases := getEnabledBases()

	for _, base := range bases {
		maxWidth := 0
		for _, value := range values {
			str := toString(value.number, base)
			if len(str) > maxWidth {
				maxWidth = len(str)
			}
		}
		widths[base] = maxWidth
	}

	return widths
}

// return list of bases to display based on command-line flags
func getEnabledBases() []int {
	bases := []int{10} // Always show decimal
	if options.showHex {
		bases = append(bases, 16)
	}
	if options.showBinary {
		bases = append(bases, 2)
	}
	if options.showOctal {
		bases = append(bases, 8)
	}
	return bases
}

func (s *Stack) print() {
	widths := maxWidths(s.values)
	bases := getEnabledBases()

	for i := len(s.values) - 1; i >= 0; i-- {
		value := s.values[i]
		separator := ""

		// Print each enabled base
		for _, base := range bases {
			// Skip binary and octal for non-integral numbers
			// For hex, skip non-integral numbers unless showHexFloat is enabled
			if base != 10 && !value.number.isIntegral() {
				if base != 16 || !options.showHexFloat {
					continue
				}
			}

			str := toString(value.number, base)
			fmt.Printf("%s%*s", separator, widths[base], str)
			separator = "  " // Two spaces between columns
		}

		// Add units if present
		if !value.units.empty() {
			fmt.Printf(" %s", value.units.String())
		}

		fmt.Println()
	}
}

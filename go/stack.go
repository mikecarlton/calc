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

func (s *Stack) reduce(op string) {
	if len(s.values) < 2 {
		die("Not enough arguments for reduction operation '@%s', exiting", op)
	}

	// Reduce all values on the stack using the given operation
	// Start with the bottom value and apply the operation left-to-right
	result := s.values[0]
	for i := 1; i < len(s.values); i++ {
		result = result.binaryOp(op, s.values[i])
	}

	// Clear the stack and push the result
	s.values = []Value{result}
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

// ColumnWidths tracks integer and fractional part widths for alignment
type ColumnWidths struct {
	integerWidth    int // width of integer part (before decimal point)
	fractionalWidth int // width of fractional part (including decimal point)
}

// return max widths for all enabled base columns, separating integer and fractional parts
func maxWidths(values []Value) map[int]ColumnWidths {
	widths := make(map[int]ColumnWidths)
	bases := getEnabledBases()

	for _, base := range bases {
		maxIntWidth := 0
		maxFracWidth := 0

		for _, value := range values {
			// Skip this base if not applicable to this value type
			if base != 10 && !value.number.isIntegral() {
				if base != 16 || !options.showHexFloat {
					continue
				}
			}

			str := toString(value.number, base)
			intPart, fracPart := splitNumber(str)

			if len(intPart) > maxIntWidth {
				maxIntWidth = len(intPart)
			}
			if len(fracPart) > maxFracWidth {
				maxFracWidth = len(fracPart)
			}
		}

		widths[base] = ColumnWidths{
			integerWidth:    maxIntWidth,
			fractionalWidth: maxFracWidth,
		}
	}

	return widths
}

// splitNumber splits a number string into integer and fractional parts
// For hex floats like "0x1.92p+06", splits at the decimal point
// For regular decimals like "100.5", splits at the decimal point
// Returns (integerPart, fractionalPart) where fractionalPart includes the decimal point
func splitNumber(str string) (string, string) {
	// Handle hex floating point format (e.g., "0x1.92p+06")
	if strings.HasPrefix(str, "0x") && strings.Contains(str, ".") {
		parts := strings.SplitN(str, ".", 2)
		return parts[0], "." + parts[1]
	}
	// Handle regular decimal format
	if strings.Contains(str, ".") {
		parts := strings.SplitN(str, ".", 2)
		return parts[0], "." + parts[1]
	}
	// Integer - no fractional part
	return str, ""
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
			intPart, fracPart := splitNumber(str)
			colWidth := widths[base]

			// Print with units digit alignment: right-align integer part, left-align fractional part
			fmt.Printf("%s%*s%s", separator, colWidth.integerWidth, intPart, fracPart)

			// Pad fractional part to maintain column alignment
			padding := colWidth.fractionalWidth - len(fracPart)
			if padding > 0 {
				fmt.Printf("%*s", padding, "")
			}

			separator = "  " // Two spaces between columns
		}

		// Add units if present
		if !value.units.empty() {
			fmt.Printf(" %s", value.units.String())
		}

		fmt.Println()
	}
}

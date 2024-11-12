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

// return max width of integer portion and max width of entire number
func maxWidths(values []Value) (int, int) {
	maxIntWidth := 0
	maxFrac := 0

	for _, value := range values {
		str := toString(value.number, 10)
		parts := strings.Split(str, ".")
		if len(parts[0]) > maxIntWidth {
			maxIntWidth = len(parts[0])
		}

		if len(parts) > 1 && len(parts[1]) > maxFrac {
			maxFrac = len(parts[1]) + 1
		}
	}

	return maxIntWidth, maxFrac
}

func (s *Stack) print() {
	maxIntWidth, maxFrac := maxWidths(s.values)
	for i := len(s.values) - 1; i >= 0; i-- {
		num := toString(s.values[i].number, 10)
		parts := strings.Split(num, ".")
		fmt.Printf("%*s", maxIntWidth, parts[0])
		if len(parts) > 1 {
			fmt.Printf(".%s", parts[1])
		}

		if !s.values[i].units.empty() {
			pad := 0
			if len(parts) == 1 {
				pad = maxFrac
			}
			fmt.Printf("%*s %s", pad, "", s.values[i].units.String())
		}

		fmt.Println()
	}
}

// Copyright 2024 Mike Carlton
// Released under terms of the MIT License:
//   http://www.opensource.org/licenses/mit-license.php

package main

import (
	"fmt"
	"os"
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
		die("Not enough arguments for binary operation '%s', exiting\n", op)
	}

	s.push(left.binaryOp(op, right))
}

func (s *Stack) unaryOp(op string) {
	value, err := s.pop()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Not enough arguments for unary operation '%s', exiting\n", op)
		os.Exit(1)
	}

	s.push(value.unaryOp(op))
}

func (s *Stack) apply(units Units) {
	value, err := s.pop()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Not enough arguments for '%s', exiting\n", units)
		os.Exit(1)
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

func (s *Stack) print() {
	for i := len(s.values) - 1; i >= 0; i-- {
		fmt.Println(s.values[i])
	}
}

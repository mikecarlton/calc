// Copyright 2024 Mike Carlton
// Released under terms of the MIT License:
//   http://www.opensource.org/licenses/mit-license.php

package main

import (
	"fmt"
	"os"
)

type Stack struct {
	values []Value
}

func newStack() *Stack {
	return &Stack{values: []Value{}}
}

func (s *Stack) binaryOp(op string) {
	right, _ := s.pop()
	left, err := s.pop()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Not enough arguments for '%s', exiting\n", op)
		os.Exit(1)
	}

	s.push(left.binaryOp(right, op))
}

func (s *Stack) unaryOp(op string) {
	value, err := s.pop()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Not enough arguments for '%s', exiting\n", op)
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

	value.units = units
	s.push(value)
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

func (s *Stack) size() int {
	return len(s.values)
}

func (s *Stack) print() {
	for i := len(s.values) - 1; i >= 0; i-- {
		fmt.Println(s.values[i])
	}
}

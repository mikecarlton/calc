// Copyright 2024 Mike Carlton
// Released under terms of the MIT License:
//   http://www.opensource.org/licenses/mit-license.php

package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) == 1 {
		usage()
		os.Exit(1)
	}

	args := scanOptions(os.Args[1:])

	// TODO: maybe keep history and print where error occurred
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "Error: %v, exiting\n", r)
			os.Exit(1)
		}
	}()

	stack := newStack()
	for _, arg := range args {
		if options.trace {
			fmt.Printf("[%s] %s\n", stack.oneline(), arg)
		}
		if num, ok := parseNumber(arg); ok {
			stack.push(Value{number: num})
		} else if time, ok := parseTime(arg); ok {
			stack.push(Value{number: time})
		} else if units, ok := parseUnits(arg); ok {
			stack.apply(units)
		} else if operator, ok := OPERATOR[arg]; ok {
			if operator.unary {
				stack.unaryOp(arg)
			} else {
				stack.binaryOp(arg)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Unrecognized argument '%s', exiting\n", arg)
			os.Exit(1)
		}
	}

	stack.print()
}

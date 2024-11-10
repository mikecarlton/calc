// Copyright 2024 Mike Carlton
// Released under terms of the MIT License:
//   http://www.opensource.org/licenses/mit-license.php

package main

import (
	"fmt"
	"os"
	"strings"
)

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

type Aliases map[string]string

func unalias(aliases Aliases, input string) string {
	if name, ok := aliases[input]; ok {
		return name
	}
	return input
}

func main() {
	if len(os.Args) == 1 {
		usage()
		os.Exit(1)
	}

	args := scanOptions(os.Args[1:])

	// TODO: maybe keep history and print where error occurred
	defer func() {
		if r := recover(); r != nil {
			die("Error: %v, exiting", r)
		}
	}()

	stack := newStack()
	for _, arg := range args {
		parts := strings.Fields(arg)
		for _, part := range parts {
			if options.trace {
				fmt.Printf("[%s] %s\n", stack.oneline(), part)
			}
			if num, ok := parseNumber(part); ok {
				stack.push(Value{number: num})
			} else if time, ok := parseTime(part); ok {
				stack.push(Value{number: time})
			} else if units, ok := parseUnits(part); ok {
				stack.apply(units)
			} else if stackOp, ok := STACKOP[unalias(STACKALIAS, part)]; ok {
				stackOp(stack)
			} else if operator, ok := OPERATOR[unalias(OPALIAS, part)]; ok {
				if operator.unary {
					stack.unaryOp(part)
				} else {
					stack.binaryOp(part)
				}
			} else {
				die("Unrecognized argument '%s', exiting", part)
			}
		}
	}

	stack.print()
}

// Copyright 2024 Mike Carlton
// Released under terms of the MIT License:
//   http://www.opensource.org/licenses/mit-license.php

package main

import (
	"bufio"
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

var CONSTANTS = map[string]*Number{
	"pi": Pi,
}

// readStdinValues reads lines from stdin and extracts values
func readStdinValues(stack *Stack) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		
		var value string
		if options.column != 0 {
			// Extract specific column
			fields := strings.Fields(line)
			if len(fields) == 0 {
				continue
			}
			
			var index int
			if options.column > 0 {
				// Positive column number (1-based)
				index = options.column - 1
				if index >= len(fields) {
					continue // Skip lines that don't have enough columns
				}
			} else {
				// Negative column number (count from end)
				index = len(fields) + options.column
				if index < 0 {
					continue // Skip lines that don't have enough columns
				}
			}
			value = fields[index]
		} else {
			// Use entire line
			value = line
		}
		
		// Try to parse the value
		if num, ok := parseNumber(value); ok {
			stack.push(Value{number: num})
		} else if base60, ok := parseBase60(value); ok {
			// Base-60 input with ':' - just a regular number
			stack.push(Value{number: base60})
		} else if constant, ok := CONSTANTS[value]; ok {
			stack.push(Value{number: constant})
		} else {
			// Skip non-numeric values
			if options.trace {
				fmt.Fprintf(os.Stderr, "Skipping non-numeric value: '%s'\n", value)
			}
		}
	}
	
	if err := scanner.Err(); err != nil {
		die("Error reading stdin: %v", err)
	}
}

func main() {
	// TODO: maybe keep history and print where error occurred
	defer func() {
		if r := recover(); r != nil {
			die("Error: %v, exiting", r)
		}
	}()

	args := scanOptions(os.Args[1:])

	// Check if we should read from stdin
	stdinAvailable := false
	if stat, err := os.Stdin.Stat(); err == nil {
		stdinAvailable = (stat.Mode() & os.ModeCharDevice) == 0
	}

	// If no arguments and no stdin, show usage
	if len(args) == 0 && !stdinAvailable {
		usage()
		os.Exit(1)
	}

	stack := newStack()
	
	// Read from stdin first if available
	if stdinAvailable {
		readStdinValues(stack)
	}

	// Process command line arguments
	for _, arg := range args {
		parts := strings.Fields(arg)
		for _, part := range parts {
			if options.trace {
				fmt.Printf("[%s] %s\n", stack.oneline(), part)
			}
			if num, ok := parseNumber(part); ok {
				stack.push(Value{number: num})
			} else if base60, ok := parseBase60(part); ok {
				// Base-60 input with ':' - just a regular number
				stack.push(Value{number: base60})
			} else if constant, ok := CONSTANTS[part]; ok {
				stack.push(Value{number: constant})
			} else if units, ok := parseUnits(part); ok {
				stack.apply(units)
			} else if stackOp, ok := STACKOP[unalias(STACKALIAS, part)]; ok {
				stackOp(stack)
			} else if strings.HasPrefix(part, "@") && len(part) > 1 {
				// Stack reduction operation (@+, @*, etc.)
				opName := unalias(OPALIAS, part[1:])
				if operator, ok := OPERATOR[opName]; ok && !operator.unary {
					stack.reduce(opName)
				} else {
					die("Invalid reduction operation '%s', exiting", part)
				}
			} else if operator, ok := OPERATOR[unalias(OPALIAS, part)]; ok {
				if operator.unary {
					stack.unaryOp(unalias(OPALIAS, part))
				} else {
					stack.binaryOp(unalias(OPALIAS, part))
				}
			} else {
				die("Unrecognized argument '%s', exiting", part)
			}
		}
	}

	// Show statistics if requested
	if options.showStats {
		stack.printStats()
	} else {
		stack.print()
	}
}

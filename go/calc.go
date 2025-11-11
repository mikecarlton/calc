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

// Color utility functions for terminal output
func green(text string) string {
	return fmt.Sprintf("\033[32m%s\033[0m", text)
}

func yellow(text string) string {
	return fmt.Sprintf("\033[33m%s\033[0m", text)
}

func red(text string) string {
	return fmt.Sprintf("\033[31m%s\033[0m", text)
}

func die(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "%s\n", red(message))
	os.Exit(1)
}

type Aliases map[string]string

func unalias(aliases Aliases, input string) string {
	if name, ok := aliases[input]; ok {
		return name
	}
	return input
}

var CONSTANTS = map[string]Value{
	"e": { // e = 2.718281828459045235
		number: newRationalNumber(2_718_281_828_459_045_235, 1_000_000_000_000_000_000),
	},
	"pi": {
		number: Pi,
	},
	"G": { // g = 9.80665 m/s²
		number: newRationalNumber(980_665, 100_000),
		units: Unit{Length: UnitPower{BaseUnit{name: "m", dimension: Length, factor: newNumber(1)}, 1},
			Time: UnitPower{BaseUnit{name: "s", dimension: Time, factor: newNumber(1)}, -2}},
	},
	"c": { // c = 299,792,458 m/s
		number: newNumber(299_792_458),
		units: Unit{Length: UnitPower{BaseUnit{name: "m", dimension: Length, factor: newNumber(1)}, 1},
			Time: UnitPower{BaseUnit{name: "s", dimension: Time, factor: newNumber(1)}, -1}},
	},
	"acre": { // 1 acre = 66x660 ft = 43,560 square feet
		number: newNumber(43_560),
		units: Unit{
			Length: UnitPower{BaseUnit{name: "ft", dimension: Length, factor: newRationalNumber(254*12, 10_000)}, 2},
		},
	},
	"hectare": { // 1 hectare = 100x100 m = 10,000 square meters
		number: newNumber(10_000),
		units: Unit{
			Length: UnitPower{BaseUnit{name: "m", dimension: Length, factor: newNumber(1)}, 2},
		},
	},
}

// readStdinValues reads lines from stdin and extracts values
func readStdinValues() []string {
	var values []string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

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
			values = append(values, fields[index])
		} else {
			// Use entire line
			values = append(values, line)
		}
	}

	if err := scanner.Err(); err != nil {
		die("Error reading stdin: %v", err)
	}

	return values
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

	generatePrefixedUnits()

	stack := newStack()

	// Read from stdin first if available
	var stdinValues []string
	if stdinAvailable {
		stdinValues = readStdinValues()
	}

	// Combine stdin values with command line arguments
	allArgs := append(stdinValues, args...)

	// Process all arguments
	for _, arg := range allArgs {
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
			} else if ipv4, ok := parseIPv4(part); ok {
				// IPv4 address input - convert to integer
				stack.push(Value{number: ipv4})
			} else if constant, ok := CONSTANTS[part]; ok {
				stack.push(constant)
			} else if units, ok := parseUnits(part); ok {
				stack.apply(units)
			} else if stackOp, ok := STACKOP[unalias(STACKALIAS, part)]; ok {
				stackOp(stack)
			} else if strings.HasPrefix(part, "@") && len(part) > 1 {
				// Check if this is a stock symbol (@aapl) or stack reduction (@+, @*, etc.)
				potentialSymbol := part[1:]
				opName := unalias(OPALIAS, potentialSymbol)
				if operator, ok := OPERATOR[opName]; ok && !operator.unary {
					// Stack reduction operation (@+, @*, etc.)
					stack.reduce(opName)
				} else {
					// Stock symbol lookup (@aapl, @msft, etc.)
					if value, err := getStockQuote(potentialSymbol); err != nil {
						fmt.Fprintf(os.Stderr, "ticker symbol '%s' not found: %v\n", potentialSymbol, err)
					} else if value != nil {
						stack.push(*value)
					}
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

	// Show stock quotes if any were fetched
	showQuotes()

	// Show statistics if requested
	if options.showStats {
		stack.printStats()
	} else if options.oneline {
		fmt.Println(stack.oneline())
	} else {
		stack.print()
	}
}

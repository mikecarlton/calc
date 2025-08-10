// Copyright 2024 Mike Carlton
// Released under terms of the MIT License:
//   http://www.opensource.org/licenses/mit-license.php

package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Options struct {
	group        bool
	trace        bool
	precision    int
	showBinary   bool
	showHex      bool
	showHexFloat bool
	showOctal    bool
	showIPv4     bool
	showRational bool
	date         string
	column       int
	showStats    bool
}

var options = Options{
	precision: 4,
}

func heredoc(text string) string {
	lines := strings.Split(strings.TrimRight(text, " \t\n"), "\n")

	// Find the minimum leading whitespace for non-empty lines
	minIndent := -1
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			leadingSpaces := len(line) - len(strings.TrimLeft(line, " "))
			if minIndent == -1 || leadingSpaces < minIndent {
				minIndent = leadingSpaces
			}
		}
	}

	// Remove the minimum leading whitespace from each line
	for i, line := range lines {
		if len(line) >= minIndent {
			lines[i] = line[minIndent:]
		}
	}

	return strings.Join(lines, "\n")
}

func usage() {
	fmt.Printf("%s\n", heredoc(fmt.Sprintf(`
        Usage: calc [OPTIONS | ARGUMENTS]
        Options:
          -t         Trace operations
          -b         Show binary representation of integers
          -o         Show octal representation of integers
          -x         Show hex representation of integers
          -X         Show hex representation of integers and floating point numbers
          -i         Show IPv4 representation of integers
          -r         Show rational representation (numerator/denominator)
          -g         Use ',' to group decimal numbers, '_' to group others
          -s         Show statistics summary
          -c Integer Column to extract from lines on stdin (negative counts from end)
          -p Integer Set display precision for floating point number (default: %d)
          -D Date    Date for currency conversion rates (e.g. 2022-01-01)
          -h         Show extended help
	`, options.precision)))

	/*
	   -q         Do not show stack at finish
	   -o         Show final stack on one line
	   -v         Verbose output (repeat for additional output)
	   -u         Show units
	*/
}

func doHelp() {
	usage()

	fmt.Printf("%s\n", heredoc(`
        Constants:
          pi
    `))

	fmt.Printf("%s\n", heredoc(`
        Numbers:
          Decimal integers
          Hexadecimal integers (leading 0x or 0X)
          Octal integers (leading 0o or 0O)
          Binary integers (leading 0b or 0B)

          Decimal floating point numbers (with optional exponent: [eE][-+]?[0-9]+)
          Hexadecimal floating point numbers (leading 0x or 0X and optional exponent: [pP][-+]?[0-9]+)
            The exponent is number of bits, in decimal

          Numbers can have a final binary magnitude factor (KMGTPEZY) for
          kilo-, mega-, giga-, tera-, peta-, exa-, zetta- or yotta-byte
    `))

	fmt.Printf("%s\n", heredoc(`
        Stack Operations:
          x: exchange top 2 elements of the stack
          d: duplicate top element of the stack (aliased as dup)
          p: pop top element off of the stack (aliased as pop)

        Stack statistics: (append '!' to replace the stack):
          mini: push minimum value onto stack
          max:  push maximum value onto stack
          mean: push mean (average) value onto stack
          size: push stack size onto stack
    `))

	fmt.Printf("%s\n", heredoc(`
        Binary numerical operations (prepend with '@' to reduce the stack):
          + - * /
          *   (aliased as . and •)
          %   (modulo, dimensionless values only)
          **  (aliased as pow, dimensionless values only)

        Unary numerical operations:
          n     (number: remove any units)
          chs   (change sign)
          t     (truncate to integer)
          log   (natural log)
          log10 (base 10 log)
          log2  (base 2 log)
		  mask  (IPv4 mask)
		  r     (reciprocal)

        Bitwise operations (integers only):
          &     (bitwise AND, prepend with '@' to reduce the stack)
          |     (bitwise OR, prepend with '@' to reduce the stack)
          ^     (bitwise XOR, prepend with '@' to reduce the stack)
          <<    (left shift)
          >>    (right shift)
          ~     (bitwise NOT/complement)
    `))

	fmt.Printf("%s\n", heredoc(`
        Units:
          Units are applied if current top of stack does not have any units
          Otherwise the current top of stack is converted to the units

          time
            seconds (s), minutes (min), hours (hr)
          length
            nanometers (nm), micrometers (um), millimeters (mm), centimeters (cm), meters (m), kilometers (km)
            inches (in), feet (ft), yards (yd), miles (mi)
          volume
            milliliters (ml), centiliters (cl), deciliters (dl), liters (l)
            fl. ounces (foz), cups (cup), pints (pt), quarts (qt), us gallons (gal)
          mass
            micrograms (ug), milligrams (mg), grams (g), kilograms (kg)
            ounces (oz), pounds (lb)
          temperature
            celsius (C or °C), delta celsius (dC)
            fahrenheit (F or °F), delta fahrenheit (dF)
          currency
            euros (eur or €), gb pounds (gbp or £), yen (yen or ¥), bitcoin (btc), us dollars (usd or $)
    `))
}

func scanOptions(args []string) []string {
	for i := 0; i < len(args); { // scan args for options, e.g. -h, -p N
		consumed := 1
		switch args[i] {
		case "-h":
			doHelp()
			os.Exit(1)
		case "-t":
			options.trace = true
		case "-s":
			options.showStats = true
		case "-g":
			options.group = true
		case "-x":
			options.showHex = true
		case "-X":
			options.showHex = true
			options.showHexFloat = true
		case "-o":
			options.showOctal = true
		case "-b":
			options.showBinary = true
		case "-i":
			options.showIPv4 = true
		case "-r":
			options.showRational = true
		case "-c":
			if i < len(args)-1 {
				if column, err := strconv.Atoi(args[i+1]); err == nil {
					options.column = column
					consumed = 2
				} else {
					fmt.Fprintf(os.Stderr, "Integer argument required for '%s', cannot parse '%s', exiting\n", args[i], args[i+1])
					os.Exit(1)
				}
			} else {
				fmt.Fprintf(os.Stderr, "Missing required argument for '%s', exiting\n", args[i])
				os.Exit(1)
			}
		case "-D":
			if i < len(args)-1 {
				options.date = args[i+1]
				consumed = 2
			} else {
				fmt.Fprintf(os.Stderr, "Missing required argument for '%s', exiting\n", args[i])
				os.Exit(1)
			}
		case "-p":
			if i < len(args)-1 {
				if precision, err := strconv.Atoi(args[i+1]); err == nil {
					options.precision = precision
					consumed = 2
				} else {
					fmt.Fprintf(os.Stderr, "Integer argument required for '%s', cannot parse '%s', exiting\n", args[i], args[i+1])
					os.Exit(1)
				}
			} else {
				fmt.Fprintf(os.Stderr, "Missing required argument for '%s', exiting\n", args[i])
				os.Exit(1)
			}
		default:
			consumed = 0
		}

		if consumed == 0 {
			i++
		} else {
			args = append(args[:i], args[i+consumed:]...) // remove the option and any argument
		}
	}

	return args
}

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
	group        string
	trace        bool
	precision    int
	showBinary   bool
	showHex      bool
	showHexFloat bool
	showOctal    bool
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
          -p Integer Set display precision for floating point number (default: %d)
          -h         Show extended help
	`, options.precision)))

	/*
	   -g         Use ',' to group decimal numbers
	   -s         Show statistics of values
	   -q         Do not show stack at finish
	   -o         Show final stack on one line
	   -D Date    Date for currency conversion rates (e.g. 2022-01-01)
	   -i         Show IPv4 representation of integers
	   -v         Verbose output (repeat for additional output)
	   -u         Show units
	*/
}

func units() {
}

func doHelp() {
	usage()

	fmt.Printf("%s\n", heredoc(`
        Numbers:
          pi
		  Decimal integers
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
    `))

	fmt.Printf("%s\n", heredoc(`
        Constants:
          pi
    `))

	fmt.Printf("%s\n", heredoc(`
        Stack Operations:
          x: exchange top 2 elements of the stack
          d: duplicate top element of the stack (aliased as dup)
          p: pop top element off of the stack (aliased as pop)
    `))

	fmt.Printf("%s\n", heredoc(`
        Binary numerical operations:
          + -
          *   (aliased as . and •)
          **  (alias pow)

        Unary numerical operations:
          chs   (change sign)
          t     (truncate to integer)
          log   (natural log)
          log10 (base 10 log)
          log2  (base 2 log)
    `))

	fmt.Printf("%s\n", heredoc(`
        Units:
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
		case "-g":
			options.group = ","
		case "-x":
			options.showHex = true
		case "-X":
			options.showHex = true
			options.showHexFloat = true
		case "-o":
			options.showOctal = true
		case "-b":
			options.showBinary = true
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

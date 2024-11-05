// Copyright 2024 Mike Carlton
// Released under terms of the MIT License:
//   http://www.opensource.org/licenses/mit-license.php

package main

import (
	"fmt"
	"os"
	"strings"
)

func doHelp(_ string) {
	os.Exit(0)
}

func usage() {
	msg := `
        Usage: calc [OPTIONS | ARGUMENTS]
        Options:
          -t         Trace operations
          -b         Show binary representation of integers
          -x         Show hex representation of integers
          -i         Show IPv4 representation of integers
          -p Integer Set display precision for floating point number (default: 2)
          -g         Use ',' to group decimal numbers
          -s         Show statistics of values
          -q         Do not show stack at finish
          -o         Show final stack on one line
          -D Date    Date for currency conversion rates (e.g. 2022-01-01)
          -v         Verbose output (repeat for additional output)
          -u         Show units
          -h         Show extended help
    `
	formattedText := strings.ReplaceAll(strings.TrimSpace(msg), "\n        ", "\n")
	fmt.Println(formattedText)
}

func main() {
	if len(os.Args) == 1 {
		usage()
		os.Exit(1)
	}

	stack := newStack()
	ops := map[string]func(arg string){
		"+":   stack.binaryOp,
		"-":   stack.binaryOp,
		"*":   stack.binaryOp,
		".":   stack.binaryOp,
		"/":   stack.binaryOp,
		"chs": stack.unaryOp,
		"-h":  doHelp,
	}

	for _, arg := range os.Args[1:] {
		if num, ok := parseNumber(arg); ok {
			stack.push(Value{number: num})
		} else if op, ok := ops[arg]; ok {
			op(arg)
			//} else if units, ok := parseUnits(arg); ok {
			//	stack.apply(units)
		} else {
			fmt.Fprintf(os.Stderr, "Unrecognized argument '%s', exiting\n", arg)
			os.Exit(1)
		}

		/*
			switch arg {
			case "+", "-", "*", ".", "/":
				stack.binaryOp(arg)
			case "chs":
				stack.unaryOp(arg)
			default:
				fmt.Fprintf(os.Stderr, "Unrecognized argument '%s', exiting\n", arg)
				os.Exit(1)
			}
		*/
	}

	stack.print()
}

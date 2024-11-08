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
	group      string
	trace      bool
	precision  int
	showBinary bool
	showHex    bool
	showOctal  bool
}

var options = Options{
	precision: 2,
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

func doHelp() {
	usage()
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

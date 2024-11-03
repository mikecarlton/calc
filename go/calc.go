// Copyright 2024 Mike Carlton
// Released under terms of the MIT License:
//   http://www.opensource.org/licenses/mit-license.php

package main

import (
	"fmt"
	"os"
)

func doHelp(_ string) {
	fmt.Printf("Usage: %s ARG(S)\n", os.Args[0])
	os.Exit(0)
}

func main() {
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
		if n, result := parseNumber(arg); result {
			v := Value{number: n}
			stack.push(v)
		} else if op, ok := ops[arg]; ok {
			op(arg)
		} else if units, result := parseUnits(arg); result {
			// apply units
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
	}

	stack.print()
}

// Copyright 2024 Mike Carlton
// Released under terms of the MIT License:
//   http://www.opensource.org/licenses/mit-license.php

package main

import (
	"os"
)

func main() {
	stack := newStack()

	for _, arg := range os.Args[1:] {
		n, result := parseNumber(arg)

		if result {
			v := Value{number: n}
			stack.push(v)
		} else {
			switch arg {
			case "+", "-", "*", "/":
				stack.binaryOp(arg)
			default:
			}
		}
	}

	stack.print()
}

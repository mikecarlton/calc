// Copyright 2024 Mike Carlton
// Released under terms of the MIT License:
//   http://www.opensource.org/licenses/mit-license.php

package main

import (
	"os"
	// "regexp"
)

func main() {
	stack := newStack()

	for _, arg := range os.Args {
		n, _ := newNumber(arg)
		v := Value{number: n}
		stack.push(v)
	}

	stack.print()
}

// Copyright 2024 Mike Carlton
// Released under terms of the MIT License:
//   http://www.opensource.org/licenses/mit-license.php

package main

type Value struct {
	number Number
	units  []Unit
}

func (v Value) String() string {
	return v.number.String()
}

// Copyright 2024 Mike Carlton
// Released under terms of the MIT License:
//   http://www.opensource.org/licenses/mit-license.php

package main

import (
	"fmt"
	"math/big"
)

// Number is a big integer or float
type Number struct {
	i *big.Int
	f *big.Float
}

func newNumber(input interface{}) (Number, error) {
	switch v := input.(type) {
	case int64:
		return Number{i: big.NewInt(v)}, nil
	case float64:
		return Number{f: big.NewFloat(v)}, nil
	case string:
		// Try parsing as an integer or float.
		if i, ok := new(big.Int).SetString(v, 10); ok {
			return Number{i: i}, nil
		} else if f, ok := new(big.Float).SetString(v); ok {
			return Number{f: f}, nil
		}
	}

	return Number{}, fmt.Errorf("unsupported type: %T", input)
}

func (n Number) Integral() bool {
	if n.i != nil {
		return true
	} else {
		return false
	}
}

func (n Number) String() string {
	if n.i != nil {
		return n.i.String()
	}
	if n.f != nil {
		return n.f.String()
	}
	return ""
}

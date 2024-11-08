// Copyright 2024 Mike Carlton
// Released under terms of the MIT License:
//   http://www.opensource.org/licenses/mit-license.php

package enumerable

func Filter[T any](slice []T, predicate func(T) bool) []T {
	filtered := make([]T, 0)
	for _, elem := range slice {
		if predicate(elem) {
			filtered = append(filtered, elem)
		}
	}
	return filtered
}

func Map[T, R any](slice []T, mapper func(T) R) []R {
	mapped := make([]R, len(slice))
	for i, elem := range slice {
		mapped[i] = mapper(elem)
	}
	return mapped
}

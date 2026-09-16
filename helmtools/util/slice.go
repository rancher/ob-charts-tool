package util

import "slices"

// FilterSlice returns a new slice containing only elements that satisfy the filter function.
// Uses slices.DeleteFunc for efficient in-place filtering after cloning.
func FilterSlice[T any](slice []T, filterFn func(T) bool) []T {
	return slices.DeleteFunc(slices.Clone(slice), func(t T) bool {
		return !filterFn(t)
	})
}

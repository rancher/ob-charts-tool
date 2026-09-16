package util

import "slices"

// FilterSlice returns a new slice containing only elements that satisfy the filter function.
// Uses slices.DeleteFunc for efficient in-place filtering after cloning.
func FilterSlice[T any](slice []T, filterFn func(T) bool) []T {
	return slices.DeleteFunc(slices.Clone(slice), func(t T) bool {
		return !filterFn(t)
	})
}

// Unique returns a new slice with duplicate elements removed, preserving the order
// of first occurrence. This is useful when you need to deduplicate a slice while
// maintaining insertion order.
func Unique[T comparable](input []T) []T {
	seen := make(map[T]struct{})
	result := make([]T, 0, len(input))

	for _, val := range input {
		if _, exists := seen[val]; !exists {
			seen[val] = struct{}{}
			result = append(result, val)
		}
	}
	return result
}

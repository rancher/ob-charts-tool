package util

func FilterSlice[T any](slice []T, filterFn func(T) bool) []T {
	filteredSlice := make([]T, 0)
	for _, element := range slice {
		if filterFn(element) {
			filteredSlice = append(filteredSlice, element)
		}
	}
	return filteredSlice
}

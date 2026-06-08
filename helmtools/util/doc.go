// Package util provides shared utilities for helmtools.
//
// # HTTP Utilities
//
// Fetch URL body:
//
//	body, err := util.GetHTTPBody(ctx, url, httpClient)
//
// # Set Operations
//
// Create and use a set:
//
//	set := util.NewSet[string]()
//	set.Add("item1")
//	set.Add("item2")
//	if set.Contains("item1") {
//		// ...
//	}
//
// # Slice Utilities
//
// Filter a slice:
//
//	evens := util.FilterSlice(numbers, func(n int) bool {
//		return n%2 == 0
//	})
package util

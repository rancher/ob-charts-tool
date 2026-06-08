// Package values provides utilities for navigating and manipulating Helm values.yaml structure.
//
// # Basic Usage
//
// Navigate to a value by dotted path:
//
//	value, found := values.NavigatePath(data, "image.tag")
//
// Navigate to a nested map:
//
//	imageMap, found := values.NavigateMap(data, "kubeRBACProxy.image")
//
// Work with subchart rules for version management:
//
//	rule := values.SubchartRule{
//		ValuesKey: "image.tag",
//		PrepareFunc: func(v string) string { return "v" + v },
//	}
//	expectedTag := rule.Apply(appVersion)
package values

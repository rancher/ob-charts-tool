// Package values provides utilities for navigating and manipulating Helm values.yaml structure.
//
// # Basic Usage
//
// Get a value by dotted path:
//
//	value, found := values.GetByPath(data, "image.tag")
//
// Get a nested map by path:
//
//	imageMap, found := values.GetMapByPath(data, "kubeRBACProxy.image")
//
// Work with subchart rules for version management:
//
//	rule := values.SubchartRule{
//		ValuesKey: "image.tag",
//		PrepareFunc: func(v string) string { return "v" + v },
//	}
//	expectedTag := rule.Apply(appVersion)
package values

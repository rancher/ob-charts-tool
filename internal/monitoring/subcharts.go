package monitoring

import (
	"fmt"
	"strings"
)

// SubchartRule defines what image tag field in values.yaml should reflect a subchart's Chart.yaml appVersion.
type SubchartRule struct {
	// ValuesKey is the dotted path in values.yaml (e.g. "image.tag" or "kubeRBACProxy.image.tag").
	ValuesKey string
	// PrepareFunc optionally transforms the appVersion before use (e.g. to prepend "v").
	// If nil, appVersion is used as-is.
	PrepareFunc func(string) string
}

// Apply returns the expected tag value for the given appVersion.
func (r SubchartRule) Apply(appVersion string) string {
	if r.PrepareFunc != nil {
		return r.PrepareFunc(appVersion)
	}
	return appVersion
}

// DefaultRules applies to subcharts with no specific entry in SubchartRules.
var DefaultRules = []SubchartRule{
	{ValuesKey: "image.tag"},
}

// SubchartRules maps normalized subchart names (without "rancher-" prefix) to their rules.
// Subcharts not listed here use DefaultRules.
var SubchartRules = map[string][]SubchartRule{
	// kube-state-metrics: both the main image and kubeRBACProxy fall back to
	// `v{.Chart.AppVersion}` in the chart's helpers template when .Values.*.image.tag
	// is empty, so Rancher patches must explicitly set both to the appVersion with a "v" prefix.
	"kube-state-metrics": {
		{ValuesKey: "image.tag", PrepareFunc: func(v string) string { return "v" + v }},
		{ValuesKey: "kubeRBACProxy.image.tag", PrepareFunc: func(v string) string { return "v" + v }},
	},
}

// SubchartsToCheck is the set of normalized subchart names (without "rancher-" prefix) whose
// image tags should be verified against their Chart.yaml appVersion.
var SubchartsToCheck = map[string]bool{
	"grafana":            true,
	"kube-state-metrics": true,
	"node-exporter":      true,
	"prometheus-adapter": true,
	"windows-exporter":   true,
}

// GetRules returns the applicable rules for a given normalized subchart name.
func GetRules(normalizedName string) []SubchartRule {
	if rules, ok := SubchartRules[normalizedName]; ok {
		return rules
	}
	return DefaultRules
}

// NormalizeName strips the "rancher-" prefix from a subchart directory name.
func NormalizeName(name string) string {
	return strings.TrimPrefix(name, "rancher-")
}

// TagMismatch describes a single discrepancy between a rule's expected tag value and what was
// found in a values.yaml map.
type TagMismatch struct {
	ValuesKey     string
	ActualValue   string
	ExpectedValue string
}

// TagMatchesExpected reports whether actual satisfies expected, allowing for appCo image tag
// conventions: a leading "v" may be absent from the actual tag, and an appCo build-revision
// suffix (e.g. "-10.11") may be appended.
//
// Examples that match:
//
//	actual="v2.10.0"      expected="v2.10.0"   → exact match
//	actual="v2.10.0-1"   expected="v2.10.0"   → revision suffix
//	actual="2.10.0-10.11" expected="v2.10.0"  → no-v + revision suffix (appCo style)
func TagMatchesExpected(actual, expected string) bool {
	norm := strings.TrimPrefix
	a := norm(actual, "v")
	e := norm(expected, "v")
	return a == e || strings.HasPrefix(a, e+"-")
}

// CheckTagsInValues inspects a parsed values.yaml map for a given subchart and appVersion,
// returning any keys whose value does not match the rule's expected tag.
// Actual values may carry an appCo build-revision suffix (e.g. "v2.10.0-1") and are still
// considered matching.
func CheckTagsInValues(normalizedName, appVersion string, values map[string]interface{}) []TagMismatch {
	var mismatches []TagMismatch
	for _, rule := range GetRules(normalizedName) {
		expected := rule.Apply(appVersion)
		actual, found := NavigateYAMLPath(values, rule.ValuesKey)
		if !found {
			mismatches = append(mismatches, TagMismatch{
				ValuesKey:     rule.ValuesKey,
				ActualValue:   "(not found)",
				ExpectedValue: expected,
			})
			continue
		}
		if !TagMatchesExpected(actual, expected) {
			mismatches = append(mismatches, TagMismatch{
				ValuesKey:     rule.ValuesKey,
				ActualValue:   actual,
				ExpectedValue: expected,
			})
		}
	}
	return mismatches
}

// ImageMapPath returns the dotted path to the containing image map for a rule.
// For "image.tag" it returns "image"; for "kubeRBACProxy.image.tag" it returns "kubeRBACProxy.image".
func (r SubchartRule) ImageMapPath() string {
	parts := strings.Split(r.ValuesKey, ".")
	if len(parts) <= 1 {
		return ""
	}
	return strings.Join(parts[:len(parts)-1], ".")
}

// NavigateYAMLPath follows a dotted key path (e.g. "image.tag") through a parsed YAML map
// and returns the string value at that path.
func NavigateYAMLPath(data map[string]interface{}, keyPath string) (string, bool) {
	parts := strings.Split(keyPath, ".")
	current := data
	for i, part := range parts {
		val, ok := current[part]
		if !ok {
			return "", false
		}
		if i == len(parts)-1 {
			if s, ok := val.(string); ok {
				return s, true
			}
			return fmt.Sprintf("%v", val), true
		}
		next, ok := val.(map[string]interface{})
		if !ok {
			return "", false
		}
		current = next
	}
	return "", false
}

// NavigateYAMLMap follows a dotted key path through a parsed YAML map and returns the
// nested map at that path. Useful for navigating to an image struct (e.g. "kubeRBACProxy.image").
func NavigateYAMLMap(data map[string]interface{}, keyPath string) (map[string]interface{}, bool) {
	parts := strings.Split(keyPath, ".")
	current := data
	for _, part := range parts {
		val, ok := current[part]
		if !ok {
			return nil, false
		}
		next, ok := val.(map[string]interface{})
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

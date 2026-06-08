package version

import (
	"strings"

	"github.com/rancher/ob-charts-tool/helmtools/values"
)

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

// VerifyTagsInValues inspects a parsed values.yaml map for a given subchart and appVersion,
// returning any keys whose value does not match the rule's expected tag.
// Actual values may carry an appCo build-revision suffix (e.g. "v2.10.0-1") and are still
// considered matching.
func VerifyTagsInValues(rules []values.SubchartRule, appVersion string, valuesMap map[string]interface{}) []TagMismatch {
	var mismatches []TagMismatch
	for _, rule := range rules {
		expected := rule.Apply(appVersion)
		actual, found := values.GetByPath(valuesMap, rule.ValuesKey)
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

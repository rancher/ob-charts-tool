package values

import "strings"

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

// ImageMapPath returns the dotted path to the containing image map for a rule.
// For "image.tag" it returns "image"; for "kubeRBACProxy.image.tag" it returns "kubeRBACProxy.image".
func (r SubchartRule) ImageMapPath() string {
	parts := strings.Split(r.ValuesKey, ".")
	if len(parts) <= 1 {
		return ""
	}
	return strings.Join(parts[:len(parts)-1], ".")
}

// GetRules returns the applicable rules for a given subchart name.
// If the name exists in ruleMap, returns those rules; otherwise returns defaults.
// This is a generic function that can be used with any rule configuration.
func GetRules(name string, ruleMap map[string][]SubchartRule, defaults []SubchartRule) []SubchartRule {
	if rules, ok := ruleMap[name]; ok {
		return rules
	}
	return defaults
}

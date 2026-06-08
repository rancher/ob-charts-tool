package values_test

import (
	"testing"

	"github.com/rancher/ob-charts-tool/helmtools/values"
)

func TestGetByPath(t *testing.T) {
	tests := []struct {
		name      string
		data      map[string]interface{}
		keyPath   string
		wantValue string
		wantFound bool
	}{
		{
			name: "simple key",
			data: map[string]interface{}{
				"tag": "v1.0.0",
			},
			keyPath:   "tag",
			wantValue: "v1.0.0",
			wantFound: true,
		},
		{
			name: "nested path",
			data: map[string]interface{}{
				"image": map[string]interface{}{
					"tag": "v2.0.0",
				},
			},
			keyPath:   "image.tag",
			wantValue: "v2.0.0",
			wantFound: true,
		},
		{
			name: "deeply nested path",
			data: map[string]interface{}{
				"kubeRBACProxy": map[string]interface{}{
					"image": map[string]interface{}{
						"tag": "v0.14.0",
					},
				},
			},
			keyPath:   "kubeRBACProxy.image.tag",
			wantValue: "v0.14.0",
			wantFound: true,
		},
		{
			name: "key not found",
			data: map[string]interface{}{
				"image": map[string]interface{}{
					"repository": "rancher/app",
				},
			},
			keyPath:   "image.tag",
			wantValue: "",
			wantFound: false,
		},
		{
			name: "path through non-map",
			data: map[string]interface{}{
				"image": "not-a-map",
			},
			keyPath:   "image.tag",
			wantValue: "",
			wantFound: false,
		},
		{
			name: "integer value",
			data: map[string]interface{}{
				"replicas": 3,
			},
			keyPath:   "replicas",
			wantValue: "3",
			wantFound: true,
		},
		{
			name: "boolean value",
			data: map[string]interface{}{
				"enabled": true,
			},
			keyPath:   "enabled",
			wantValue: "true",
			wantFound: true,
		},
		{
			name: "float value",
			data: map[string]interface{}{
				"version": 1.5,
			},
			keyPath:   "version",
			wantValue: "1.5",
			wantFound: true,
		},
		{
			name:      "nil data",
			data:      nil,
			keyPath:   "image.tag",
			wantValue: "",
			wantFound: false,
		},
		{
			name: "empty keyPath",
			data: map[string]interface{}{
				"tag": "v1.0.0",
			},
			keyPath:   "",
			wantValue: "",
			wantFound: false,
		},
		{
			name: "top-level key missing",
			data: map[string]interface{}{
				"other": "value",
			},
			keyPath:   "image.tag",
			wantValue: "",
			wantFound: false,
		},
		{
			name:      "empty data map",
			data:      map[string]interface{}{},
			keyPath:   "tag",
			wantValue: "",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValue, gotFound := values.GetByPath(tt.data, tt.keyPath)
			if gotFound != tt.wantFound {
				t.Errorf("GetByPath() found = %v, want %v", gotFound, tt.wantFound)
			}
			if gotValue != tt.wantValue {
				t.Errorf("GetByPath() value = %v, want %v", gotValue, tt.wantValue)
			}
		})
	}
}

func TestGetMapByPath(t *testing.T) {
	tests := []struct {
		name      string
		data      map[string]interface{}
		keyPath   string
		wantMap   map[string]interface{}
		wantFound bool
	}{
		{
			name: "simple map",
			data: map[string]interface{}{
				"image": map[string]interface{}{
					"tag":        "v1.0.0",
					"repository": "rancher/app",
				},
			},
			keyPath: "image",
			wantMap: map[string]interface{}{
				"tag":        "v1.0.0",
				"repository": "rancher/app",
			},
			wantFound: true,
		},
		{
			name: "nested map",
			data: map[string]interface{}{
				"kubeRBACProxy": map[string]interface{}{
					"image": map[string]interface{}{
						"tag":        "v0.14.0",
						"repository": "brancz/kube-rbac-proxy",
					},
				},
			},
			keyPath: "kubeRBACProxy.image",
			wantMap: map[string]interface{}{
				"tag":        "v0.14.0",
				"repository": "brancz/kube-rbac-proxy",
			},
			wantFound: true,
		},
		{
			name: "deeply nested map",
			data: map[string]interface{}{
				"a": map[string]interface{}{
					"b": map[string]interface{}{
						"c": map[string]interface{}{
							"value": "deep",
						},
					},
				},
			},
			keyPath: "a.b.c",
			wantMap: map[string]interface{}{
				"value": "deep",
			},
			wantFound: true,
		},
		{
			name: "key not found",
			data: map[string]interface{}{
				"image": map[string]interface{}{
					"tag": "v1.0.0",
				},
			},
			keyPath:   "other",
			wantMap:   nil,
			wantFound: false,
		},
		{
			name: "path through non-map",
			data: map[string]interface{}{
				"image": "not-a-map",
			},
			keyPath:   "image.tag",
			wantMap:   nil,
			wantFound: false,
		},
		{
			name: "intermediate path not found",
			data: map[string]interface{}{
				"image": map[string]interface{}{
					"tag": "v1.0.0",
				},
			},
			keyPath:   "image.nested.deep",
			wantMap:   nil,
			wantFound: false,
		},
		{
			name:      "nil data",
			data:      nil,
			keyPath:   "image",
			wantMap:   nil,
			wantFound: false,
		},
		{
			name: "empty keyPath",
			data: map[string]interface{}{
				"image": map[string]interface{}{
					"tag": "v1.0.0",
				},
			},
			keyPath:   "",
			wantMap:   nil,
			wantFound: false,
		},
		{
			name:      "empty data map",
			data:      map[string]interface{}{},
			keyPath:   "image",
			wantMap:   nil,
			wantFound: false,
		},
		{
			name: "leaf is not a map",
			data: map[string]interface{}{
				"image": map[string]interface{}{
					"tag": "v1.0.0",
				},
			},
			keyPath:   "image.tag",
			wantMap:   nil,
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMap, gotFound := values.GetMapByPath(tt.data, tt.keyPath)
			if gotFound != tt.wantFound {
				t.Errorf("GetMapByPath() found = %v, want %v", gotFound, tt.wantFound)
			}
			if tt.wantFound {
				if !mapsEqual(gotMap, tt.wantMap) {
					t.Errorf("GetMapByPath() map = %v, want %v", gotMap, tt.wantMap)
				}
			} else if gotMap != nil {
				t.Errorf("GetMapByPath() when not found should return nil map, got %v", gotMap)
			}
		})
	}
}

func TestGetRules(t *testing.T) {
	defaultRules := []values.SubchartRule{
		{ValuesKey: "image.tag"},
	}

	customRules := []values.SubchartRule{
		{ValuesKey: "custom.tag"},
		{ValuesKey: "other.tag"},
	}

	ruleMap := map[string][]values.SubchartRule{
		"special-chart": customRules,
	}

	tests := []struct {
		name         string
		chartName    string
		ruleMap      map[string][]values.SubchartRule
		defaults     []values.SubchartRule
		wantRules    []values.SubchartRule
		wantRuleKeys []string
	}{
		{
			name:         "chart with custom rules",
			chartName:    "special-chart",
			ruleMap:      ruleMap,
			defaults:     defaultRules,
			wantRules:    customRules,
			wantRuleKeys: []string{"custom.tag", "other.tag"},
		},
		{
			name:         "chart without custom rules uses defaults",
			chartName:    "regular-chart",
			ruleMap:      ruleMap,
			defaults:     defaultRules,
			wantRules:    defaultRules,
			wantRuleKeys: []string{"image.tag"},
		},
		{
			name:         "empty rule map uses defaults",
			chartName:    "any-chart",
			ruleMap:      map[string][]values.SubchartRule{},
			defaults:     defaultRules,
			wantRules:    defaultRules,
			wantRuleKeys: []string{"image.tag"},
		},
		{
			name:         "nil rule map uses defaults",
			chartName:    "any-chart",
			ruleMap:      nil,
			defaults:     defaultRules,
			wantRules:    defaultRules,
			wantRuleKeys: []string{"image.tag"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := values.GetRules(tt.chartName, tt.ruleMap, tt.defaults)
			if len(got) != len(tt.wantRules) {
				t.Errorf("GetRules() returned %d rules, want %d", len(got), len(tt.wantRules))
				return
			}
			for i, rule := range got {
				if rule.ValuesKey != tt.wantRuleKeys[i] {
					t.Errorf("GetRules()[%d].ValuesKey = %v, want %v", i, rule.ValuesKey, tt.wantRuleKeys[i])
				}
			}
		})
	}
}

func TestSubchartRule_Apply(t *testing.T) {
	tests := []struct {
		name       string
		rule       values.SubchartRule
		appVersion string
		want       string
	}{
		{
			name: "no PrepareFunc",
			rule: values.SubchartRule{
				ValuesKey: "image.tag",
			},
			appVersion: "2.10.0",
			want:       "2.10.0",
		},
		{
			name: "PrepareFunc adds v prefix",
			rule: values.SubchartRule{
				ValuesKey: "image.tag",
				PrepareFunc: func(v string) string {
					return "v" + v
				},
			},
			appVersion: "2.10.0",
			want:       "v2.10.0",
		},
		{
			name: "PrepareFunc transforms version",
			rule: values.SubchartRule{
				ValuesKey: "image.tag",
				PrepareFunc: func(v string) string {
					return "release-" + v
				},
			},
			appVersion: "1.0.0",
			want:       "release-1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rule.Apply(tt.appVersion)
			if got != tt.want {
				t.Errorf("SubchartRule.Apply() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSubchartRule_ImageMapPath(t *testing.T) {
	tests := []struct {
		name      string
		valuesKey string
		want      string
	}{
		{
			name:      "simple image.tag",
			valuesKey: "image.tag",
			want:      "image",
		},
		{
			name:      "nested path",
			valuesKey: "kubeRBACProxy.image.tag",
			want:      "kubeRBACProxy.image",
		},
		{
			name:      "single segment",
			valuesKey: "tag",
			want:      "",
		},
		{
			name:      "three level path",
			valuesKey: "a.b.tag",
			want:      "a.b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := values.SubchartRule{ValuesKey: tt.valuesKey}
			got := rule.ImageMapPath()
			if got != tt.want {
				t.Errorf("SubchartRule.ImageMapPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function to compare maps
func mapsEqual(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

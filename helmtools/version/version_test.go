package version

import (
	"testing"

	"github.com/rancher/ob-charts-tool/helmtools/values"
)

func TestTagMatchesExpected(t *testing.T) {
	tests := []struct {
		name     string
		actual   string
		expected string
		want     bool
	}{
		{
			name:     "exact match with v prefix",
			actual:   "v2.10.0",
			expected: "v2.10.0",
			want:     true,
		},
		{
			name:     "exact match without v prefix",
			actual:   "2.10.0",
			expected: "2.10.0",
			want:     true,
		},
		{
			name:     "actual has v, expected does not",
			actual:   "v2.10.0",
			expected: "2.10.0",
			want:     true,
		},
		{
			name:     "expected has v, actual does not",
			actual:   "2.10.0",
			expected: "v2.10.0",
			want:     true,
		},
		{
			name:     "revision suffix with v prefix",
			actual:   "v2.10.0-1",
			expected: "v2.10.0",
			want:     true,
		},
		{
			name:     "appCo style revision suffix",
			actual:   "2.10.0-10.11",
			expected: "v2.10.0",
			want:     true,
		},
		{
			name:     "revision suffix without v prefix",
			actual:   "2.10.0-1",
			expected: "2.10.0",
			want:     true,
		},
		{
			name:     "version mismatch",
			actual:   "v2.9.0",
			expected: "v2.10.0",
			want:     false,
		},
		{
			name:     "version mismatch with revision",
			actual:   "v2.9.0-1",
			expected: "v2.10.0",
			want:     false,
		},
		{
			name:     "empty strings",
			actual:   "",
			expected: "",
			want:     true,
		},
		{
			name:     "empty actual",
			actual:   "",
			expected: "v2.10.0",
			want:     false,
		},
		{
			name:     "empty expected",
			actual:   "v2.10.0",
			expected: "",
			want:     false,
		},
		{
			name:     "dash in expected version",
			actual:   "v2.10.0-alpha",
			expected: "v2.10.0",
			want:     true,
		},
		{
			name:     "multiple dashes in revision",
			actual:   "v2.10.0-10.11-beta",
			expected: "v2.10.0",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TagMatchesExpected(tt.actual, tt.expected)
			if got != tt.want {
				t.Errorf("TagMatchesExpected(%q, %q) = %v, want %v", tt.actual, tt.expected, got, tt.want)
			}
		})
	}
}

func TestVerifyTagsInValues(t *testing.T) {
	tests := []struct {
		name              string
		rules             []values.SubchartRule
		appVersion        string
		valuesMap         map[string]interface{}
		wantMismatches    int
		wantFirstMismatch *TagMismatch
	}{
		{
			name: "all tags match exactly",
			rules: []values.SubchartRule{
				{ValuesKey: "image.tag"},
			},
			appVersion: "v2.10.0",
			valuesMap: map[string]interface{}{
				"image": map[string]interface{}{
					"tag": "v2.10.0",
				},
			},
			wantMismatches: 0,
		},
		{
			name: "tag matches with revision suffix",
			rules: []values.SubchartRule{
				{ValuesKey: "image.tag"},
			},
			appVersion: "v2.10.0",
			valuesMap: map[string]interface{}{
				"image": map[string]interface{}{
					"tag": "v2.10.0-1",
				},
			},
			wantMismatches: 0,
		},
		{
			name: "tag matches without v prefix",
			rules: []values.SubchartRule{
				{ValuesKey: "image.tag"},
			},
			appVersion: "v2.10.0",
			valuesMap: map[string]interface{}{
				"image": map[string]interface{}{
					"tag": "2.10.0",
				},
			},
			wantMismatches: 0,
		},
		{
			name: "tag mismatch",
			rules: []values.SubchartRule{
				{ValuesKey: "image.tag"},
			},
			appVersion: "v2.10.0",
			valuesMap: map[string]interface{}{
				"image": map[string]interface{}{
					"tag": "v2.9.0",
				},
			},
			wantMismatches: 1,
			wantFirstMismatch: &TagMismatch{
				ValuesKey:     "image.tag",
				ActualValue:   "v2.9.0",
				ExpectedValue: "v2.10.0",
			},
		},
		{
			name: "tag not found",
			rules: []values.SubchartRule{
				{ValuesKey: "image.tag"},
			},
			appVersion:     "v2.10.0",
			valuesMap:      map[string]interface{}{},
			wantMismatches: 1,
			wantFirstMismatch: &TagMismatch{
				ValuesKey:     "image.tag",
				ActualValue:   "(not found)",
				ExpectedValue: "v2.10.0",
			},
		},
		{
			name: "multiple rules with mixed results",
			rules: []values.SubchartRule{
				{ValuesKey: "image.tag"},
				{ValuesKey: "kubeRBACProxy.image.tag"},
				{ValuesKey: "other.tag"},
			},
			appVersion: "v2.10.0",
			valuesMap: map[string]interface{}{
				"image": map[string]interface{}{
					"tag": "v2.10.0",
				},
				"kubeRBACProxy": map[string]interface{}{
					"image": map[string]interface{}{
						"tag": "v2.9.0",
					},
				},
			},
			wantMismatches: 2,
			wantFirstMismatch: &TagMismatch{
				ValuesKey:     "kubeRBACProxy.image.tag",
				ActualValue:   "v2.9.0",
				ExpectedValue: "v2.10.0",
			},
		},
		{
			name: "rule with PrepareFunc",
			rules: []values.SubchartRule{
				{
					ValuesKey: "image.tag",
					PrepareFunc: func(v string) string {
						return "v" + v
					},
				},
			},
			appVersion: "2.10.0",
			valuesMap: map[string]interface{}{
				"image": map[string]interface{}{
					"tag": "v2.10.0",
				},
			},
			wantMismatches: 0,
		},
		{
			name: "rule with PrepareFunc actual mismatch",
			rules: []values.SubchartRule{
				{
					ValuesKey: "image.tag",
					PrepareFunc: func(v string) string {
						return "v" + v
					},
				},
			},
			appVersion: "2.10.0",
			valuesMap: map[string]interface{}{
				"image": map[string]interface{}{
					"tag": "v2.9.0",
				},
			},
			wantMismatches: 1,
			wantFirstMismatch: &TagMismatch{
				ValuesKey:     "image.tag",
				ActualValue:   "v2.9.0",
				ExpectedValue: "v2.10.0",
			},
		},
		{
			name:           "nil rules",
			rules:          nil,
			appVersion:     "v2.10.0",
			valuesMap:      map[string]interface{}{},
			wantMismatches: 0,
		},
		{
			name:           "empty rules",
			rules:          []values.SubchartRule{},
			appVersion:     "v2.10.0",
			valuesMap:      map[string]interface{}{},
			wantMismatches: 0,
		},
		{
			name: "nil values map",
			rules: []values.SubchartRule{
				{ValuesKey: "image.tag"},
			},
			appVersion:     "v2.10.0",
			valuesMap:      nil,
			wantMismatches: 1,
			wantFirstMismatch: &TagMismatch{
				ValuesKey:     "image.tag",
				ActualValue:   "(not found)",
				ExpectedValue: "v2.10.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mismatches := VerifyTagsInValues(tt.rules, tt.appVersion, tt.valuesMap)
			if len(mismatches) != tt.wantMismatches {
				t.Errorf("VerifyTagsInValues() returned %d mismatches, want %d", len(mismatches), tt.wantMismatches)
			}
			if tt.wantFirstMismatch != nil && len(mismatches) > 0 {
				got := mismatches[0]
				want := tt.wantFirstMismatch
				if got.ValuesKey != want.ValuesKey || got.ActualValue != want.ActualValue || got.ExpectedValue != want.ExpectedValue {
					t.Errorf("First mismatch = %+v, want %+v", got, want)
				}
			}
		})
	}
}

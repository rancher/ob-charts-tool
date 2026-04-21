package monitoring

import (
	"testing"
)

func TestNormalizeName(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"rancher-grafana", "grafana"},
		{"rancher-kube-state-metrics", "kube-state-metrics"},
		{"rancher-node-exporter", "node-exporter"},
		{"grafana", "grafana"},                       // no prefix to strip
		{"", ""},                                     // empty string
		{"rancher-", ""},                             // only prefix
		{"rancher-rancher-foo", "rancher-foo"},       // only leading prefix stripped
		{"prometheus-adapter", "prometheus-adapter"}, // non-rancher prefix untouched
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := NormalizeName(tc.input)
			if got != tc.want {
				t.Errorf("NormalizeName(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestSubchartRule_Apply(t *testing.T) {
	cases := []struct {
		name        string
		prepareFunc func(string) string
		appVersion  string
		want        string
	}{
		{
			name:        "nil PrepareFunc returns as-is",
			prepareFunc: nil,
			appVersion:  "1.2.3",
			want:        "1.2.3",
		},
		{
			name:        "PrepareFunc prepends v",
			prepareFunc: func(v string) string { return "v" + v },
			appVersion:  "1.2.3",
			want:        "v1.2.3",
		},
		{
			name:        "nil PrepareFunc with empty version",
			prepareFunc: nil,
			appVersion:  "",
			want:        "",
		},
		{
			name:        "PrepareFunc with empty version",
			prepareFunc: func(v string) string { return "v" + v },
			appVersion:  "",
			want:        "v",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rule := SubchartRule{ValuesKey: "image.tag", PrepareFunc: tc.prepareFunc}
			got := rule.Apply(tc.appVersion)
			if got != tc.want {
				t.Errorf("SubchartRule.Apply(%q) = %q, want %q", tc.appVersion, got, tc.want)
			}
		})
	}
}

func TestGetRules(t *testing.T) {
	t.Run("kube-state-metrics returns custom rules", func(t *testing.T) {
		rules := GetRules("kube-state-metrics")
		if len(rules) != 2 {
			t.Fatalf("GetRules(kube-state-metrics) returned %d rules, want 2", len(rules))
		}
		keys := []string{rules[0].ValuesKey, rules[1].ValuesKey}
		wantKeys := map[string]bool{"image.tag": true, "kubeRBACProxy.image.tag": true}
		for _, k := range keys {
			if !wantKeys[k] {
				t.Errorf("unexpected rule key %q", k)
			}
		}
		// Both rules should have a PrepareFunc that adds "v"
		for _, r := range rules {
			got := r.Apply("2.0.0")
			if got != "v2.0.0" {
				t.Errorf("kube-state-metrics rule.Apply(2.0.0) = %q, want v2.0.0", got)
			}
		}
	})

	t.Run("grafana returns DefaultRules", func(t *testing.T) {
		rules := GetRules("grafana")
		if len(rules) != len(DefaultRules) {
			t.Fatalf("GetRules(grafana) returned %d rules, want %d (DefaultRules)", len(rules), len(DefaultRules))
		}
		if rules[0].ValuesKey != "image.tag" {
			t.Errorf("GetRules(grafana)[0].ValuesKey = %q, want image.tag", rules[0].ValuesKey)
		}
		if rules[0].PrepareFunc != nil {
			t.Errorf("GetRules(grafana)[0].PrepareFunc should be nil for default rule")
		}
	})

	t.Run("unknown subchart returns DefaultRules", func(t *testing.T) {
		rules := GetRules("some-unknown-chart")
		if len(rules) != len(DefaultRules) {
			t.Fatalf("GetRules(unknown) returned %d rules, want %d", len(rules), len(DefaultRules))
		}
	})
}

func TestNavigateYAMLPath(t *testing.T) {
	cases := []struct {
		name      string
		data      map[string]any
		keyPath   string
		wantVal   string
		wantFound bool
	}{
		{
			name:      "single level key found",
			data:      map[string]any{"tag": "v1.0"},
			keyPath:   "tag",
			wantVal:   "v1.0",
			wantFound: true,
		},
		{
			name:      "two level nested",
			data:      map[string]any{"image": map[string]any{"tag": "abc"}},
			keyPath:   "image.tag",
			wantVal:   "abc",
			wantFound: true,
		},
		{
			name: "three level nested",
			data: map[string]any{
				"a": map[string]any{
					"b": map[string]any{"c": "val"},
				},
			},
			keyPath:   "a.b.c",
			wantVal:   "val",
			wantFound: true,
		},
		{
			name:      "missing top-level key",
			data:      map[string]any{"image": map[string]any{"tag": "abc"}},
			keyPath:   "notfound",
			wantVal:   "",
			wantFound: false,
		},
		{
			name:      "missing nested key",
			data:      map[string]any{"image": map[string]any{"tag": "abc"}},
			keyPath:   "image.missing",
			wantVal:   "",
			wantFound: false,
		},
		{
			name:      "intermediate is not a map",
			data:      map[string]any{"image": "notamap"},
			keyPath:   "image.tag",
			wantVal:   "",
			wantFound: false,
		},
		{
			name:      "integer value at leaf uses fmt.Sprintf",
			data:      map[string]any{"image": map[string]any{"tag": 42}},
			keyPath:   "image.tag",
			wantVal:   "42",
			wantFound: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, found := NavigateYAMLPath(tc.data, tc.keyPath)
			if found != tc.wantFound {
				t.Errorf("NavigateYAMLPath found=%v, want %v", found, tc.wantFound)
			}
			if got != tc.wantVal {
				t.Errorf("NavigateYAMLPath val=%q, want %q", got, tc.wantVal)
			}
		})
	}
}

func TestNavigateYAMLMap(t *testing.T) {
	cases := []struct {
		name    string
		data    map[string]any
		keyPath string
		wantOk  bool
		wantKey string // a key that should exist in the returned map, if ok
	}{
		{
			name:    "single level returns nested map",
			data:    map[string]any{"image": map[string]any{"tag": "v1"}},
			keyPath: "image",
			wantOk:  true,
			wantKey: "tag",
		},
		{
			name: "two levels",
			data: map[string]any{
				"a": map[string]any{
					"b": map[string]any{"c": "x"},
				},
			},
			keyPath: "a.b",
			wantOk:  true,
			wantKey: "c",
		},
		{
			name:    "missing key",
			data:    map[string]any{"image": map[string]any{"tag": "v1"}},
			keyPath: "notfound",
			wantOk:  false,
		},
		{
			name:    "non-map value returns false",
			data:    map[string]any{"image": "stringvalue"},
			keyPath: "image",
			wantOk:  false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := NavigateYAMLMap(tc.data, tc.keyPath)
			if ok != tc.wantOk {
				t.Errorf("NavigateYAMLMap ok=%v, want %v", ok, tc.wantOk)
			}
			if tc.wantOk && tc.wantKey != "" {
				if _, exists := got[tc.wantKey]; !exists {
					t.Errorf("NavigateYAMLMap result missing expected key %q", tc.wantKey)
				}
			}
		})
	}
}

func TestCheckTagsInValues(t *testing.T) {
	cases := []struct {
		name           string
		normalizedName string
		appVersion     string
		values         map[string]any
		wantMismatches int
		wantKey        string // ValuesKey of first mismatch, if any
		wantActual     string // ActualValue of first mismatch, if any
		wantExpected   string // ExpectedValue of first mismatch, if any
	}{
		{
			name:           "grafana: matching tag",
			normalizedName: "grafana",
			appVersion:     "10.0.0",
			values:         map[string]any{"image": map[string]any{"tag": "10.0.0"}},
			wantMismatches: 0,
		},
		{
			name:           "grafana: mismatched tag",
			normalizedName: "grafana",
			appVersion:     "10.0.0",
			values:         map[string]any{"image": map[string]any{"tag": "9.0.0"}},
			wantMismatches: 1,
			wantKey:        "image.tag",
			wantActual:     "9.0.0",
			wantExpected:   "10.0.0",
		},
		{
			name:           "grafana: missing tag key",
			normalizedName: "grafana",
			appVersion:     "10.0.0",
			values:         map[string]any{"image": map[string]any{}},
			wantMismatches: 1,
			wantKey:        "image.tag",
			wantActual:     "(not found)",
			wantExpected:   "10.0.0",
		},
		{
			name:           "grafana: empty values map",
			normalizedName: "grafana",
			appVersion:     "10.0.0",
			values:         map[string]any{},
			wantMismatches: 1,
			wantActual:     "(not found)",
		},
		{
			name:           "kube-state-metrics: both tags correct",
			normalizedName: "kube-state-metrics",
			appVersion:     "2.10.0",
			values: map[string]any{
				"image": map[string]any{"tag": "v2.10.0"},
				"kubeRBACProxy": map[string]any{
					"image": map[string]any{"tag": "v2.10.0"},
				},
			},
			wantMismatches: 0,
		},
		{
			name:           "kube-state-metrics: missing v prefix on image.tag",
			normalizedName: "kube-state-metrics",
			appVersion:     "2.10.0",
			values: map[string]any{
				"image": map[string]any{"tag": "2.10.0"},
				"kubeRBACProxy": map[string]any{
					"image": map[string]any{"tag": "v2.10.0"},
				},
			},
			wantMismatches: 1,
			wantKey:        "image.tag",
			wantActual:     "2.10.0",
			wantExpected:   "v2.10.0",
		},
		{
			name:           "kube-state-metrics: both tags wrong",
			normalizedName: "kube-state-metrics",
			appVersion:     "2.10.0",
			values: map[string]any{
				"image": map[string]any{"tag": "v1.0.0"},
				"kubeRBACProxy": map[string]any{
					"image": map[string]any{"tag": "v1.0.0"},
				},
			},
			wantMismatches: 2,
		},
		{
			name:           "node-exporter: uses DefaultRules, matching",
			normalizedName: "node-exporter",
			appVersion:     "1.5.0",
			values:         map[string]any{"image": map[string]any{"tag": "1.5.0"}},
			wantMismatches: 0,
		},
		{
			name:           "unknown subchart: uses DefaultRules",
			normalizedName: "nonexistent-chart",
			appVersion:     "1.0.0",
			values:         map[string]any{"image": map[string]any{"tag": "1.0.0"}},
			wantMismatches: 0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mismatches := CheckTagsInValues(tc.normalizedName, tc.appVersion, tc.values)
			if len(mismatches) != tc.wantMismatches {
				t.Errorf("CheckTagsInValues returned %d mismatches, want %d: %+v", len(mismatches), tc.wantMismatches, mismatches)
				return
			}
			if tc.wantMismatches == 1 {
				m := mismatches[0]
				if tc.wantKey != "" && m.ValuesKey != tc.wantKey {
					t.Errorf("mismatch.ValuesKey = %q, want %q", m.ValuesKey, tc.wantKey)
				}
				if tc.wantActual != "" && m.ActualValue != tc.wantActual {
					t.Errorf("mismatch.ActualValue = %q, want %q", m.ActualValue, tc.wantActual)
				}
				if tc.wantExpected != "" && m.ExpectedValue != tc.wantExpected {
					t.Errorf("mismatch.ExpectedValue = %q, want %q", m.ExpectedValue, tc.wantExpected)
				}
			}
		})
	}
}

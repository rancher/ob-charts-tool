package monitoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
			assert.Equal(t, tc.want, got, "NormalizeName(%q)", tc.input)
		})
	}
}

func TestTagMatchesExpected(t *testing.T) {
	cases := []struct {
		actual   string
		expected string
		want     bool
	}{
		{"v2.10.0", "v2.10.0", true},           // exact match
		{"v2.10.0-1", "v2.10.0", true},         // appCo single-digit revision
		{"v2.10.0-14", "v2.10.0", true},        // appCo multi-digit revision
		{"10.0.0-3.14", "10.0.0", true},        // non-v-prefixed with revision
		{"2.17.0-10.11", "v2.17.0", true},      // appCo: no-v actual, v-prefixed expected
		{"2.17.0", "v2.17.0", true},            // appCo: no-v actual, v-prefixed expected, no revision
		{"v2.10.0", "v2.10.0-1", false},        // expected has suffix, actual does not
		{"v2.11.0", "v2.10.0", false},          // different version
		{"0.20.1-16.14", "v2.17.0", false},     // unrelated version
		{"v2.10.0-alpha", "v2.10.0", true},     // non-numeric suffix also accepted
		{"v2.10.0something", "v2.10.0", false}, // suffix without dash not accepted
		{"", "", true},                         // both empty
		{"v2.10.0-1", "v2.10", false},          // partial prefix not accepted
	}
	for _, tc := range cases {
		t.Run(tc.actual+"_vs_"+tc.expected, func(t *testing.T) {
			got := TagMatchesExpected(tc.actual, tc.expected)
			assert.Equal(t, tc.want, got, "TagMatchesExpected(%q, %q)", tc.actual, tc.expected)
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
			assert.Equal(t, tc.want, got, "SubchartRule.Apply(%q)", tc.appVersion)
		})
	}
}

func TestGetRules(t *testing.T) {
	t.Run("kube-state-metrics returns custom rules", func(t *testing.T) {
		rules := GetRules("kube-state-metrics")
		assert.Len(t, rules, 1, "GetRules(kube-state-metrics)")
		assert.Equal(t, "image.tag", rules[0].ValuesKey)
		assert.Equal(t, "v2.0.0", rules[0].Apply("2.0.0"))
	})

	t.Run("grafana returns DefaultRules", func(t *testing.T) {
		rules := GetRules("grafana")
		assert.Len(t, rules, len(DefaultRules), "GetRules(grafana)")
		assert.Equal(t, "image.tag", rules[0].ValuesKey)
		assert.Nil(t, rules[0].PrepareFunc, "default rule should have nil PrepareFunc")
	})

	t.Run("unknown subchart returns DefaultRules", func(t *testing.T) {
		rules := GetRules("some-unknown-chart")
		assert.Len(t, rules, len(DefaultRules))
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
			assert.Equal(t, tc.wantFound, found, "NavigateYAMLPath found")
			assert.Equal(t, tc.wantVal, got, "NavigateYAMLPath value")
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
			assert.Equal(t, tc.wantOk, ok, "NavigateYAMLMap ok")
			if tc.wantOk && tc.wantKey != "" {
				assert.Contains(t, got, tc.wantKey, "NavigateYAMLMap result should contain key")
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
			name:           "grafana: appCo revision suffix accepted",
			normalizedName: "grafana",
			appVersion:     "10.0.0",
			values:         map[string]any{"image": map[string]any{"tag": "10.0.0-3.14"}},
			wantMismatches: 0,
		},
		{
			name:           "kube-state-metrics: appCo no-v tag with revision accepted",
			normalizedName: "kube-state-metrics",
			appVersion:     "2.17.0",
			values: map[string]any{
				"image": map[string]any{"tag": "2.17.0-10.11"},
			},
			wantMismatches: 0,
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
			name:           "kube-state-metrics: completely wrong image.tag version",
			normalizedName: "kube-state-metrics",
			appVersion:     "2.10.0",
			values: map[string]any{
				"image": map[string]any{"tag": "1.0.0"},
			},
			wantMismatches: 1,
			wantKey:        "image.tag",
			wantActual:     "1.0.0",
			wantExpected:   "v2.10.0",
		},
		{
			name:           "kube-state-metrics: both tags correct with v prefix",
			normalizedName: "kube-state-metrics",
			appVersion:     "2.10.0",
			values: map[string]any{
				"image": map[string]any{"tag": "v2.10.0"},
			},
			wantMismatches: 0,
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
			assert.Len(t, mismatches, tc.wantMismatches, "CheckTagsInValues mismatches: %+v", mismatches)
			if tc.wantMismatches == 1 {
				m := mismatches[0]
				if tc.wantKey != "" {
					assert.Equal(t, tc.wantKey, m.ValuesKey)
				}
				if tc.wantActual != "" {
					assert.Equal(t, tc.wantActual, m.ActualValue)
				}
				if tc.wantExpected != "" {
					assert.Equal(t, tc.wantExpected, m.ExpectedValue)
				}
			}
		})
	}
}

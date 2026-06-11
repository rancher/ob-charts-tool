package values_test

import (
	"testing"

	"github.com/rancher/ob-charts-tool/internal/values"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeName(t *testing.T) {
	cases := []struct {
		input       string
		want        string
		description string
	}{
		{"rancher-grafana", "grafana", "standard rancher-prefixed chart"},
		{"rancher-kube-state-metrics", "kube-state-metrics", "rancher-prefixed chart with hyphens"},
		{"rancher-node-exporter", "node-exporter", "rancher-prefixed chart"},
		{"grafana", "grafana", "no prefix to strip"},
		{"", "", "empty string"},
		{"rancher-", "", "only prefix"},
		{"rancher-rancher-foo", "rancher-foo", "only leading prefix stripped"},
		{"prometheus-adapter", "prometheus-adapter", "non-rancher prefix untouched"},
		{"Rancher-grafana", "Rancher-grafana", "case-sensitive: capital R not stripped"},
		{"RANCHER-grafana", "RANCHER-grafana", "case-sensitive: all caps not stripped"},
		{"rancher-grafana-", "grafana-", "trailing hyphen preserved"},
		{"rancher--grafana", "-grafana", "double hyphen after prefix"},
	}
	for _, tc := range cases {
		name := tc.input
		if name == "" {
			name = "empty_string"
		}
		t.Run(name, func(t *testing.T) {
			got := values.NormalizeName(tc.input)
			assert.Equal(t, tc.want, got, "NormalizeName(%q) - %s", tc.input, tc.description)
		})
	}
}

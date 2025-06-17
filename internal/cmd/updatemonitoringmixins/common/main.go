package common

import (
	"github.com/rancher/ob-charts-tool/internal/cmd/updatemonitoringmixins/types"
	"gopkg.in/yaml.v3"
	"strings"
)

// LiteralStr is a custom type to represent a literal block string
type LiteralStr string

func (l LiteralStr) MarshalYAML() (interface{}, error) {
	return yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: string(l),
		Style: yaml.LiteralStyle, // This is the key part to make it a |- block
		Tag:   "!!str",
	}, nil
}

func YamlStrRepr(v interface{}, indent int) (string, error) {
	var b strings.Builder
	encoder := yaml.NewEncoder(&b)
	encoder.SetIndent(2)
	err := encoder.Encode(v)
	if err != nil {
		return "", err
	}

	yamlStr := b.String()
	// Indent the YAML string by the specified amount
	// This emulates the textwrap.indent in Python
	lines := strings.Split(yamlStr, "\n")
	for i, line := range lines {
		if line != "" || i < len(lines)-1 { // Only indent non-empty lines or not the very last line
			lines[i] = strings.Repeat(" ", indent) + line
		}
	}

	// Join the lines back into a single string
	return strings.Join(lines, "\n"), nil
}

func SetDefaultMaxK8s(ds types.DashboardSource) types.DashboardSource {
	if ds.GetMaxKubernetes() == "" {
		// Equal to: https://github.com/prometheus-community/helm-charts/blob/0b60795bb66a21cd368b657f0665d67de3e49da9/charts/kube-prometheus-stack/hack/sync_grafana_dashboards.py#L326
		ds.SetMaxKubernetes("9.9.9-9")
	}

	return ds
}

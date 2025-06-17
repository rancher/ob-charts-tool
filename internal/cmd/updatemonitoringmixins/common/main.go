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

func YamlStrRepr(v interface{}, indent int, escape bool) (string, error) {
	var b strings.Builder
	encoder := yaml.NewEncoder(&b)
	encoder.SetIndent(indent)
	err := encoder.Encode(v)
	if err != nil {
		return "", err
	}

	yamlStr := b.String()
	if escape {
		yamlStr = escapeHelm(yamlStr)
	}

	return yamlStr, nil
}

func escapeHelm(s string) string {
	s = strings.ReplaceAll(s, "{{", "{{`{{")
	s = strings.ReplaceAll(s, "}}", "}}`}}")
	s = strings.ReplaceAll(s, "{{`{{", "{{`{{`}}")
	s = strings.ReplaceAll(s, "}}`}}", "{{`}}`}}")
	return s
}

func SetDefaultMaxK8s(kv types.KatesVersions) types.KatesVersions {
	if kv.GetMaxKubernetes() == "" {
		// Equal to: https://github.com/prometheus-community/helm-charts/blob/0b60795bb66a21cd368b657f0665d67de3e49da9/charts/kube-prometheus-stack/hack/sync_grafana_dashboards.py#L326
		kv.SetMaxKubernetes("9.9.9-9")
	}

	return kv
}

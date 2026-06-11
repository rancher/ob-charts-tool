package config

import "github.com/rancher/ob-charts-tool/helmtools/values"

// SubchartRules maps normalized subchart names to their tag rules.
// This is specific to kube-prometheus-stack chart verification.
var SubchartRules = map[string][]values.SubchartRule{
	"kube-state-metrics": {
		{ValuesKey: "image.tag", PrepareFunc: func(v string) string { return "v" + v }},
	},
}

// DefaultRules applies to subcharts with no specific entry in SubchartRules.
var DefaultRules = []values.SubchartRule{
	{ValuesKey: "image.tag"},
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

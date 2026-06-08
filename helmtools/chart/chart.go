package chart

import (
	"fmt"

	"github.com/rancher/ob-charts-tool/helmtools/util"
	"gopkg.in/yaml.v3"
)

// ParseChartYAML parses Chart.yaml bytes into a Chart struct.
func ParseChartYAML(data []byte) (*Chart, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("chart data cannot be empty")
	}
	var chart Chart
	err := yaml.Unmarshal(data, &chart)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Chart.yaml: %w", err)
	}
	return &chart, nil
}

// FindDependencies extracts the dependencies from a chart, filtering out "crds".
// Returns an empty slice if the chart has no dependencies.
func FindDependencies(chart *Chart) []Dependency {
	if chart == nil || len(chart.Dependencies) == 0 {
		return nil
	}

	return util.FilterSlice(chart.Dependencies, func(dep Dependency) bool {
		return dep.Name != "crds"
	})
}

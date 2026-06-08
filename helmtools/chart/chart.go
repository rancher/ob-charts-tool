package chart

import (
	"fmt"

	"github.com/rancher/ob-charts-tool/helmtools/util"
	"gopkg.in/yaml.v3"
)

// ParseChartYAML parses Chart.yaml bytes into a Chart struct.
func ParseChartYAML(data []byte) (*Chart, error) {
	var chart Chart
	err := yaml.Unmarshal(data, &chart)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Chart.yaml: %w", err)
	}
	return &chart, nil
}

// FetchChartYAML fetches Chart.yaml from a URL and parses it.
func FetchChartYAML(url string) (*Chart, error) {
	body, err := util.GetHTTPBody(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Chart.yaml from %s: %w", url, err)
	}
	return ParseChartYAML(body)
}

// FindDependencies extracts the dependencies from a chart, filtering out "crds".
// Returns an empty slice if the chart has no dependencies.
func FindDependencies(chart *Chart) []ChartDependency {
	if chart == nil || len(chart.Dependencies) == 0 {
		return nil
	}

	return util.FilterSlice(chart.Dependencies, func(dep ChartDependency) bool {
		return dep.Name != "crds"
	})
}

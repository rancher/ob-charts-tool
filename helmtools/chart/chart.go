package chart

import (
	"errors"
	"fmt"

	"github.com/rancher/ob-charts-tool/helmtools/util"
	"gopkg.in/yaml.v3"
)

// Dependency represents a Helm chart dependency.
type Dependency struct {
	Name       string `yaml:"name"`
	Version    string `yaml:"version"`
	Repository string `yaml:"repository"`
}

// Metadata contains basic chart metadata.
type Metadata struct {
	Name       string `yaml:"name"`
	Version    string `yaml:"version"`
	AppVersion string `yaml:"appVersion"`
}

// Chart represents a Helm Chart.yaml structure.
type Chart struct {
	Metadata     `yaml:",inline"`
	Dependencies []Dependency `yaml:"dependencies"`
}

// ParseChartYAML parses Chart.yaml bytes into a Chart struct.
func ParseChartYAML(data []byte) (*Chart, error) {
	if len(data) == 0 {
		return nil, errors.New("chart data cannot be empty")
	}
	var chart Chart
	err := yaml.Unmarshal(data, &chart)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Chart.yaml: %w", err)
	}
	return &chart, nil
}

// FindDependencies extracts the dependencies from a chart, filtering out "crds".
func FindDependencies(chart *Chart) []Dependency {
	if chart == nil || len(chart.Dependencies) == 0 {
		return []Dependency{}
	}

	return util.FilterSlice(chart.Dependencies, func(dep Dependency) bool {
		return dep.Name != "crds"
	})
}

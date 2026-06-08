package chart

// ChartDependency represents a Helm chart dependency.
type ChartDependency struct {
	Name       string `yaml:"name"`
	Version    string `yaml:"version"`
	Repository string `yaml:"repository"`
}

// ChartMetadata contains basic chart metadata.
type ChartMetadata struct {
	Version    string `yaml:"version"`
	AppVersion string `yaml:"appVersion"`
}

// Chart represents a Helm Chart.yaml structure.
type Chart struct {
	ChartMetadata `yaml:",inline"`
	Dependencies  []ChartDependency `yaml:"dependencies"`
}

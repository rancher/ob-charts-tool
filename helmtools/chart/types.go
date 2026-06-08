package chart

// Dependency represents a Helm chart dependency.
type Dependency struct {
	Name       string `yaml:"name"`
	Version    string `yaml:"version"`
	Repository string `yaml:"repository"`
}

// Metadata contains basic chart metadata.
type Metadata struct {
	Version    string `yaml:"version"`
	AppVersion string `yaml:"appVersion"`
}

// Chart represents a Helm Chart.yaml structure.
type Chart struct {
	Metadata     `yaml:",inline"`
	Dependencies []Dependency `yaml:"dependencies"`
}

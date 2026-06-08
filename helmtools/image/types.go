package image

// Image represents a container image with registry, repository, and tag.
type Image struct {
	Registry   string `yaml:"registry"`
	Repository string `yaml:"repository"`
	Tag        string `yaml:"tag"`
}

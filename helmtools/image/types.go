package image

// Image represents a container image with registry, repository, and tag.
type Image struct {
	Registry   string `yaml:"registry"`
	Repository string `yaml:"repository"`
	Tag        string `yaml:"tag"`
}

// ImageReference represents a container image with metadata about where it was found.
//
//nolint:revive // ImageReference is intentional - distinguishes from base Image type
type ImageReference struct {
	Image   Image
	Sources []string // Chart sources where this image was found (e.g., "chart-name:1.0.0")
	OS      string   // "linux" or "windows" (detected from image name)
}

// FullImage returns the complete image reference as a string.
// Examples: "registry/repository:tag" or "repository:tag"
func (ref ImageReference) FullImage() string {
	img := ref.Image
	if img.Registry != "" {
		return img.Registry + "/" + img.Repository + ":" + img.Tag
	}
	return img.Repository + ":" + img.Tag
}

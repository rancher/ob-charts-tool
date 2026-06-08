package image

import "strings"

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
	OS      string   // Primary OS: "linux" or "windows" (first value from OSList)
	OSList  []string // All supported OS values (e.g., ["windows", "linux"] for multi-OS images)
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

// SupportsOS checks if the image supports the given operating system.
// This is compatible with rancher/rancher's multi-OS image handling where
// an image with os: "windows,linux" supports both Windows and Linux.
func (ref ImageReference) SupportsOS(os string) bool {
	os = strings.ToLower(os)
	for _, supportedOS := range ref.OSList {
		if strings.ToLower(supportedOS) == os {
			return true
		}
	}
	return false
}

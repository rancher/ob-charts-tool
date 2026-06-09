package image

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

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

// ExtractImagesWithSources extracts images from values.yaml and tracks their source.
// The source parameter identifies where the values came from (e.g., "chart-name:1.0.0").
// Returns a map of image string -> ImageReference with aggregated sources.
func ExtractImagesWithSources(valuesData []byte, source string, defaultTag string) (map[string]*ImageReference, error) {
	var root yaml.Node
	err := yaml.Unmarshal(valuesData, &root)
	if err != nil {
		return nil, fmt.Errorf("failed to parse values.yaml: %w", err)
	}

	refs := make(map[string]*ImageReference)
	extractImagesWithSource(&root, source, defaultTag, refs)
	return refs, nil
}

// extractImagesWithSource recursively traverses a YAML node tree and tracks sources.
func extractImagesWithSource(node *yaml.Node, source string, defaultTag string, refs map[string]*ImageReference) {
	if node == nil {
		return
	}

	// Handle DocumentNode by processing its content
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		extractImagesWithSource(node.Content[0], source, defaultTag, refs)
		return
	}

	// Process MappingNode (key-value pairs)
	if node.Kind != yaml.MappingNode {
		return
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		if keyNode.Kind == yaml.ScalarNode && strings.HasSuffix(strings.ToLower(keyNode.Value), "image") {
			var img Image
			if err := valueNode.Decode(&img); err == nil && img.Repository != "" {
				// Set default tag if empty
				if img.Tag == "" && defaultTag != "" {
					img.Tag = defaultTag
				}

				// Extract explicit OS field if present
				osList := extractOSField(valueNode)

				// Build full image string as key
				fullImage := buildFullImage(img)

				// Add or update ImageReference
				if ref, exists := refs[fullImage]; exists {
					// Add source if not already present
					if !contains(ref.Sources, source) {
						ref.Sources = append(ref.Sources, source)
					}
				} else {
					// Create new reference
					var primaryOS string
					var supportedOSList []string

					if osList == nil {
						// No OS field found - fall back to name-based detection
						primaryOS = detectOSFromName(img)
						supportedOSList = []string{primaryOS}
					} else if len(osList) > 0 {
						// OS field found with valid values - use them
						primaryOS = osList[0]
						supportedOSList = osList
					} else {
						// OS field found but all values invalid - use empty OSList
						// This makes SupportsOS() return false for all OS queries
						primaryOS = "unknown"
						supportedOSList = []string{}
					}

					refs[fullImage] = &ImageReference{
						Image:   img,
						Sources: []string{source},
						OS:      primaryOS,
						OSList:  supportedOSList,
					}
				}
			}
		}

		// Recursively process nested structures
		extractImagesWithSource(valueNode, source, defaultTag, refs)
	}
}

// buildFullImage constructs the full image reference string.
func buildFullImage(img Image) string {
	if img.Registry != "" {
		return img.Registry + "/" + img.Repository + ":" + img.Tag
	}
	return img.Repository + ":" + img.Tag
}

// extractOSField extracts the OS field from an image YAML node.
// Returns a list of all OS values (e.g., ["windows", "linux"] for "windows,linux").
// Returns nil if not found or invalid.
func extractOSField(node *yaml.Node) []string {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}

	// Look for "os" field in the mapping
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		if keyNode.Kind == yaml.ScalarNode && strings.ToLower(keyNode.Value) == "os" {
			if valueNode.Kind == yaml.ScalarNode {
				osValue := strings.TrimSpace(valueNode.Value)
				// Handle comma-separated values (e.g., "windows,linux")
				osList := []string{} // Initialize as empty slice (not nil)
				for _, os := range strings.Split(osValue, ",") {
					os = strings.TrimSpace(strings.ToLower(os))
					// Only include valid OS values
					if os == "windows" || os == "linux" {
						osList = append(osList, os)
					}
				}
				return osList // Returns empty slice if no valid values found
			}
			return nil
		}
	}

	return nil
}

// detectOSFromName detects the operating system based on image name patterns.
func detectOSFromName(img Image) string {
	fullImage := strings.ToLower(buildFullImage(img))
	if strings.Contains(fullImage, "windows") ||
		strings.Contains(fullImage, "nanoserver") ||
		strings.Contains(fullImage, "windowsservercore") {
		return "windows"
	}
	return "linux"
}

// contains checks if a string slice contains a value.
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// MergeImageSources merges multiple image reference maps into one.
// Sources are deduplicated and aggregated per image.
func MergeImageSources(maps ...map[string]*ImageReference) map[string]*ImageReference {
	result := make(map[string]*ImageReference)

	for _, m := range maps {
		for key, ref := range m {
			if existing, exists := result[key]; exists {
				// Merge sources
				for _, source := range ref.Sources {
					if !contains(existing.Sources, source) {
						existing.Sources = append(existing.Sources, source)
					}
				}
			} else {
				// Copy the reference
				result[key] = &ImageReference{
					Image:   ref.Image,
					Sources: append([]string{}, ref.Sources...), // Copy slice
					OS:      ref.OS,
					OSList:  append([]string{}, ref.OSList...), // Copy slice
				}
			}
		}
	}

	return result
}

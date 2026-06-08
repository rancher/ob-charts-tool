package image

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

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
			if err := valueNode.Decode(&img); err == nil {
				// Set default tag if empty
				if img.Tag == "" && defaultTag != "" {
					img.Tag = defaultTag
				}

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
					refs[fullImage] = &ImageReference{
						Image:   img,
						Sources: []string{source},
						OS:      detectOS(img),
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

// detectOS detects the operating system based on image name patterns.
func detectOS(img Image) string {
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
				}
			}
		}
	}

	return result
}

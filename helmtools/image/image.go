package image

import (
	"fmt"
	"regexp"

	"github.com/rancher/ob-charts-tool/helmtools/util"
	"go.yaml.in/yaml/v3"
)

// Image represents a container image with registry, repository, and tag.
type Image struct {
	Registry   string `yaml:"registry"`
	Repository string `yaml:"repository"`
	Tag        string `yaml:"tag"`
}

// ExtractImages recursively extracts all image definitions from a parsed values.yaml.
// It uses pattern matching to find keys ending in "image" and decodes their values.
// If an image has an empty tag and defaultTag is provided, it uses defaultTag.
func ExtractImages(valuesData []byte, defaultTag string) (util.Set[Image], error) {
	var root yaml.Node
	err := yaml.Unmarshal(valuesData, &root)
	if err != nil {
		return util.NewSet[Image](), fmt.Errorf("failed to parse values.yaml: %w", err)
	}

	images := util.NewSet[Image]()
	extractImagesFromNode(&root, defaultTag, &images)
	return images, nil
}

var imageKeyPattern = regexp.MustCompile(`(?i)^(.+)?image$`)

// extractImagesFromNode recursively traverses a YAML node tree looking for image definitions.
func extractImagesFromNode(node *yaml.Node, defaultTag string, images *util.Set[Image]) {
	if node == nil {
		return
	}

	// Handle DocumentNode by processing its content
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		extractImagesFromNode(node.Content[0], defaultTag, images)
		return
	}

	// Process MappingNode (key-value pairs)
	if node.Kind != yaml.MappingNode {
		return
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		if keyNode.Kind == yaml.ScalarNode && imageKeyPattern.MatchString(keyNode.Value) {
			var img Image
			if err := valueNode.Decode(&img); err == nil {
				// Set default tag if empty
				if img.Tag == "" && defaultTag != "" {
					img.Tag = defaultTag
				}
				images.Add(img)
			}
		}

		// Recursively process nested structures
		extractImagesFromNode(valueNode, defaultTag, images)
	}
}

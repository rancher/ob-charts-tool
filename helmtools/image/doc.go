// Package image provides utilities for extracting container images from Helm values.yaml files.
//
// # Basic Usage
//
// Extract all images from a values.yaml:
//
//	images, err := image.ExtractImages(valuesData, "")
//
// Extract images with a default tag:
//
//	images, err := image.ExtractImages(valuesData, "v1.0.0")
//
// Extract images from rendered templates:
//
//	images := image.ExtractImagesFromTemplates(renderedChart)
//
// Extract images with source tracking:
//
//	refs, err := image.ExtractImagesWithSources(valuesData, "chart-name:1.0.0", "")
//	for _, ref := range refs {
//	    fmt.Printf("%s from %v\n", ref.FullImage(), ref.Sources)
//	}
//
// Merge images from multiple charts:
//
//	refs1, _ := image.ExtractImagesWithSources(data1, "chart-a:1.0.0", "")
//	refs2, _ := image.ExtractImagesWithSources(data2, "chart-b:2.0.0", "")
//	merged := image.MergeImageSources(refs1, refs2)
//
// The extraction uses heuristic pattern matching to find keys ending in "image"
// and decodes their structure (repository, tag, etc.).
package image

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
// The extraction uses heuristic pattern matching to find keys ending in "image"
// and decodes their structure (repository, tag, etc.).
package image

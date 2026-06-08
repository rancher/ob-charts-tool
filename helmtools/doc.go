// Package helmtools provides utilities for working with Helm charts.
//
// This package contains subpackages for common Helm operations:
//   - chart: Parse and fetch Helm Chart.yaml files
//   - git: Query Git repositories for Helm chart tags and versions
//   - image: Extract container images from Helm values.yaml files
//   - upstream: Work with upstream Helm chart repositories (Prometheus, Grafana)
//   - values: Navigate and manipulate Helm values.yaml structure
//   - version: Version comparison utilities
//   - util: Shared utilities (HTTP, sets, slices)
//
// # Basic Usage
//
// Fetch and parse a Chart.yaml:
//
//	client := chart.NewClient(nil) // uses http.DefaultClient
//	chart, err := client.FetchChartYAML(ctx, "https://example.com/Chart.yaml")
//
// Find Git tags for a chart:
//
//	found, tags, err := git.FindMatchingTags(ctx, repoURL, "prometheus-")
//
// Extract images from values.yaml:
//
//	images, err := image.ExtractImages(valuesData, "v1.0.0")
package helmtools

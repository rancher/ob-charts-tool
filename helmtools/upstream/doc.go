// Package upstream provides utilities for working with upstream Helm chart repositories.
//
// Currently supports:
//   - Prometheus Community charts
//   - Grafana charts
//
// # Basic Usage
//
// Identify which repository a chart belongs to:
//
//	repo := upstream.IdentifyRepository("grafana")
//
// Build URLs for chart files:
//
//	chartURL := upstream.BuildChartYAMLURL("kube-prometheus-stack", commitHash)
//	valuesURL := upstream.BuildValuesYAMLURL("kube-prometheus-stack", commitHash)
//
// The built URLs point to raw GitHub content.
package upstream

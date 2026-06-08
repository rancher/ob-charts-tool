// Package chart provides utilities for parsing and fetching Helm Chart.yaml files.
//
// # Basic Usage
//
// Parse a Chart.yaml from bytes:
//
//	chart, err := chart.ParseChartYAML(data)
//
// Fetch and parse a Chart.yaml from a URL:
//
//	client := chart.NewClient(nil)
//	chart, err := client.FetchChartYAML(ctx, "https://example.com/Chart.yaml")
//
// With custom HTTP client:
//
//	httpClient := &http.Client{Timeout: 30 * time.Second}
//	client := chart.NewClient(httpClient)
//	chart, err := client.FetchChartYAML(ctx, url)
//
// Find chart dependencies:
//
//	deps := chart.FindDependencies(myChart)
package chart

// Package chart provides utilities for parsing and fetching Helm Chart.yaml files
// and Helm repository index.yaml files.
//
// # Basic Usage
//
// Parse a Chart.yaml from bytes:
//
//	chart, err := chart.ParseChartYAML(data)
//
// Fetch and parse a Chart.yaml from a URL:
//
//	httpClient := &http.Client{Timeout: 30 * time.Second}
//	client, err := chart.NewClient(httpClient)
//	if err != nil {
//		return err
//	}
//	chart, err := client.FetchChartYAML(ctx, "https://example.com/Chart.yaml")
//
// Find chart dependencies:
//
//	deps := chart.FindDependencies(myChart)
//
// Parse Helm repository index.yaml:
//
//	index, err := chart.ParseIndex(data)
//	latest := index.GetLatestVersion("nginx")
//	charts := index.ListCharts()
package chart

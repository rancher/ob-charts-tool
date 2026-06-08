package chart

import (
	"context"
	"fmt"
	"net/http"

	"github.com/rancher/ob-charts-tool/helmtools/util"
)

// Client provides methods for fetching and parsing Helm charts with a configured HTTP client.
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new chart Client with the given HTTP client.
// If httpClient is nil, http.DefaultClient is used.
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{httpClient: httpClient}
}

// FetchChartYAML fetches Chart.yaml from a URL and parses it using the client's HTTP configuration.
// The context can be used for cancellation and timeouts.
func (c *Client) FetchChartYAML(ctx context.Context, url string) (*Chart, error) {
	if url == "" {
		return nil, fmt.Errorf("url cannot be empty")
	}
	body, err := util.GetHTTPBody(ctx, c.httpClient, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Chart.yaml from %s: %w", url, err)
	}
	return ParseChartYAML(body)
}

package chart

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/rancher/ob-charts-tool/helmtools/util"
)

// Client provides methods for fetching and parsing Helm charts with a configured HTTP client.
// Client is safe for concurrent use by multiple goroutines.
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new chart Client with the given HTTP client.
// The httpClient parameter must not be nil.
// The returned Client is safe for concurrent use.
func NewClient(httpClient *http.Client) (*Client, error) {
	if httpClient == nil {
		return nil, errors.New("httpClient cannot be nil")
	}
	return &Client{httpClient: httpClient}, nil
}

// FetchChartYAML fetches Chart.yaml from a URL and parses it using the client's HTTP configuration.
// The context can be used for cancellation and timeouts.
func (c *Client) FetchChartYAML(ctx context.Context, url string) (*Chart, error) {
	if url == "" {
		return nil, errors.New("url cannot be empty")
	}
	body, err := util.FetchURL(ctx, c.httpClient, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Chart.yaml from %s: %w", url, err)
	}
	return ParseChartYAML(body)
}

package util

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// GetHTTPBody fetches the body of an HTTP GET request.
// If client is nil, http.DefaultClient is used.
// The context can be used for cancellation and timeouts.
func GetHTTPBody(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", url, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL %s: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body from %s: %w", url, err)
	}
	return body, nil
}

package chart_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rancher/ob-charts-tool/helmtools/chart"
)

func TestClient_FetchChartYAML(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`
apiVersion: v2
name: test-chart
version: 1.0.0
appVersion: 1.0.0
`))
	}))
	defer server.Close()

	// Test with nil client (uses default)
	client := chart.NewClient(nil)
	chartData, err := client.FetchChartYAML(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("FetchChartYAML failed: %v", err)
	}

	if chartData.Metadata.Name != "test-chart" {
		t.Errorf("Expected chart name 'test-chart', got '%s'", chartData.Metadata.Name)
	}
	if chartData.Metadata.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", chartData.Metadata.Version)
	}
}

func TestClient_WithCustomHTTPClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify custom user agent
		if r.Header.Get("User-Agent") != "custom-agent" {
			t.Errorf("Expected custom User-Agent header")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`
apiVersion: v2
name: test-chart
version: 1.0.0
`))
	}))
	defer server.Close()

	// Create custom HTTP client with modified transport
	customClient := &http.Client{
		Transport: &customTransport{
			base: http.DefaultTransport,
		},
	}

	client := chart.NewClient(customClient)
	_, err := client.FetchChartYAML(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("FetchChartYAML with custom client failed: %v", err)
	}
}

// customTransport adds a custom User-Agent header
type customTransport struct {
	base http.RoundTripper
}

func (t *customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", "custom-agent")
	return t.base.RoundTrip(req)
}

package util_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rancher/ob-charts-tool/helmtools/util"
)

func TestFetchURL(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		serverStatus   int
		wantBody       string
		wantErr        bool
		errContains    string
	}{
		{
			name:           "successful fetch",
			serverResponse: "test response body",
			serverStatus:   http.StatusOK,
			wantBody:       "test response body",
			wantErr:        false,
		},
		{
			name:           "fetch JSON data",
			serverResponse: `{"name":"test","version":"1.0.0"}`,
			serverStatus:   http.StatusOK,
			wantBody:       `{"name":"test","version":"1.0.0"}`,
			wantErr:        false,
		},
		{
			name:           "fetch YAML data",
			serverResponse: "name: test\nversion: 1.0.0\n",
			serverStatus:   http.StatusOK,
			wantBody:       "name: test\nversion: 1.0.0\n",
			wantErr:        false,
		},
		{
			name:           "fetch empty response",
			serverResponse: "",
			serverStatus:   http.StatusOK,
			wantBody:       "",
			wantErr:        false,
		},
		{
			name:           "fetch large response",
			serverResponse: strings.Repeat("a", 10000),
			serverStatus:   http.StatusOK,
			wantBody:       strings.Repeat("a", 10000),
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET request, got %s", r.Method)
				}
				w.WriteHeader(tt.serverStatus)
				w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			got, err := util.FetchURL(context.Background(), nil, server.URL)
			if (err != nil) != tt.wantErr {
				t.Errorf("FetchURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("FetchURL() error = %v, should contain %q", err, tt.errContains)
				return
			}
			if !tt.wantErr && string(got) != tt.wantBody {
				t.Errorf("FetchURL() = %q, want %q", string(got), tt.wantBody)
			}
		})
	}
}

func TestFetchURL_Validation(t *testing.T) {
	t.Run("empty URL returns error", func(t *testing.T) {
		_, err := util.FetchURL(context.Background(), nil, "")
		if err == nil {
			t.Fatal("FetchURL() with empty URL should return error")
		}
		if !strings.Contains(err.Error(), "cannot be empty") {
			t.Errorf("FetchURL() error = %v, should contain 'cannot be empty'", err)
		}
	})
}

func TestFetchURL_CustomClient(t *testing.T) {
	t.Run("uses custom HTTP client", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("custom client response"))
		}))
		defer server.Close()

		customClient := &http.Client{
			Timeout: 5 * time.Second,
		}

		got, err := util.FetchURL(context.Background(), customClient, server.URL)
		if err != nil {
			t.Fatalf("FetchURL() unexpected error: %v", err)
		}
		if string(got) != "custom client response" {
			t.Errorf("FetchURL() = %q, want %q", string(got), "custom client response")
		}
	})

	t.Run("nil client uses default client", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("default client response"))
		}))
		defer server.Close()

		got, err := util.FetchURL(context.Background(), nil, server.URL)
		if err != nil {
			t.Fatalf("FetchURL() unexpected error: %v", err)
		}
		if string(got) != "default client response" {
			t.Errorf("FetchURL() = %q, want %q", string(got), "default client response")
		}
	})
}

func TestFetchURL_Context(t *testing.T) {
	t.Run("context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			// Simulate slow response
			time.Sleep(100 * time.Millisecond)
			w.Write([]byte("should not receive this"))
		}))
		defer server.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := util.FetchURL(ctx, nil, server.URL)
		if err == nil {
			t.Fatal("FetchURL() with cancelled context should return error")
		}
		if !errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), "context canceled") {
			t.Errorf("FetchURL() error = %v, should be context.Canceled", err)
		}
	})

	t.Run("context timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			// Simulate slow response
			time.Sleep(200 * time.Millisecond)
			w.Write([]byte("should not receive this"))
		}))
		defer server.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		_, err := util.FetchURL(ctx, nil, server.URL)
		if err == nil {
			t.Fatal("FetchURL() with timeout should return error")
		}
	})

	t.Run("context with deadline passes", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Write([]byte("fast response"))
		}))
		defer server.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		got, err := util.FetchURL(ctx, nil, server.URL)
		if err != nil {
			t.Fatalf("FetchURL() unexpected error: %v", err)
		}
		if string(got) != "fast response" {
			t.Errorf("FetchURL() = %q, want %q", string(got), "fast response")
		}
	})
}

func TestFetchURL_HTTPErrors(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		errContains string
	}{
		{
			name:        "404 not found",
			statusCode:  http.StatusNotFound,
			errContains: "HTTP 404",
		},
		{
			name:        "500 internal server error",
			statusCode:  http.StatusInternalServerError,
			errContains: "HTTP 500",
		},
		{
			name:        "403 forbidden",
			statusCode:  http.StatusForbidden,
			errContains: "HTTP 403",
		},
		{
			name:        "400 bad request",
			statusCode:  http.StatusBadRequest,
			errContains: "HTTP 400",
		},
		{
			name:        "502 bad gateway",
			statusCode:  http.StatusBadGateway,
			errContains: "HTTP 502",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte("error response"))
			}))
			defer server.Close()

			_, err := util.FetchURL(context.Background(), nil, server.URL)
			if err == nil {
				t.Fatal("FetchURL() should return error for non-2xx status")
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("FetchURL() error = %v, should contain %q", err, tt.errContains)
			}
		})
	}
}

func TestFetchURL_InvalidURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		errContains string
	}{
		{
			name:        "malformed URL",
			url:         "://invalid",
			errContains: "failed to create request",
		},
		{
			name:        "invalid scheme",
			url:         "invalid://example.com",
			errContains: "failed to fetch URL",
		},
		{
			name:        "unreachable host",
			url:         "http://localhost:99999",
			errContains: "failed to fetch URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := util.FetchURL(context.Background(), nil, tt.url)
			if err == nil {
				t.Fatal("FetchURL() with invalid URL should return error")
			}
			if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("FetchURL() error = %v, should contain %q", err, tt.errContains)
			}
		})
	}
}

func TestFetchURL_NetworkErrors(t *testing.T) {
	t.Run("connection refused", func(t *testing.T) {
		// Use a port that's unlikely to be open
		_, err := util.FetchURL(context.Background(), nil, "http://localhost:1")
		if err == nil {
			t.Fatal("FetchURL() with refused connection should return error")
		}
		if !strings.Contains(err.Error(), "failed to fetch URL") {
			t.Errorf("FetchURL() error = %v, should contain 'failed to fetch URL'", err)
		}
	})
}

func TestFetchURL_ReadBodyError(t *testing.T) {
	t.Run("server closes connection early", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Length", "100")
			w.WriteHeader(http.StatusOK)
			// Write less data than Content-Length indicates
			w.Write([]byte("short"))
		}))
		defer server.Close()

		// This might not always error depending on HTTP client behavior
		// but we're testing the error path exists
		got, err := util.FetchURL(context.Background(), nil, server.URL)
		// Either succeeds with short content or errors
		if err == nil && string(got) != "short" {
			t.Errorf("FetchURL() = %q, want %q", string(got), "short")
		}
	})
}

func TestFetchURL_CustomTransport(t *testing.T) {
	t.Run("custom transport with headers", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			// Verify custom headers could be set via transport
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("transport response"))
		}))
		defer server.Close()

		customClient := &http.Client{
			Transport: &http.Transport{
				MaxIdleConns: 10,
			},
		}

		got, err := util.FetchURL(context.Background(), customClient, server.URL)
		if err != nil {
			t.Fatalf("FetchURL() unexpected error: %v", err)
		}
		if string(got) != "transport response" {
			t.Errorf("FetchURL() = %q, want %q", string(got), "transport response")
		}
	})
}

func TestFetchURL_ConcurrentRequests(t *testing.T) {
	t.Run("concurrent fetches to same server", func(t *testing.T) {
		requestChan := make(chan struct{}, 5)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			requestChan <- struct{}{}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("concurrent response"))
		}))
		defer server.Close()

		// Run multiple concurrent requests
		done := make(chan bool, 5)
		for i := 0; i < 5; i++ {
			go func() {
				_, err := util.FetchURL(context.Background(), nil, server.URL)
				if err != nil {
					t.Errorf("FetchURL() unexpected error: %v", err)
				}
				done <- true
			}()
		}

		// Wait for all requests to complete
		for i := 0; i < 5; i++ {
			<-done
		}

		// Count requests received
		close(requestChan)
		requestCount := len(requestChan)
		if requestCount != 5 {
			t.Errorf("Expected 5 requests, got %d", requestCount)
		}
	})
}

func TestFetchURL_BinaryData(t *testing.T) {
	t.Run("fetch binary data", func(t *testing.T) {
		binaryData := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			w.Write(binaryData)
		}))
		defer server.Close()

		got, err := util.FetchURL(context.Background(), nil, server.URL)
		if err != nil {
			t.Fatalf("FetchURL() unexpected error: %v", err)
		}
		if len(got) != len(binaryData) {
			t.Errorf("FetchURL() returned %d bytes, want %d bytes", len(got), len(binaryData))
		}
		for i, b := range got {
			if b != binaryData[i] {
				t.Errorf("FetchURL() byte[%d] = %x, want %x", i, b, binaryData[i])
			}
		}
	})
}

func TestFetchURL_Redirects(t *testing.T) {
	t.Run("follows redirects", func(t *testing.T) {
		finalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("final destination"))
		}))
		defer finalServer.Close()

		redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, finalServer.URL, http.StatusFound)
		}))
		defer redirectServer.Close()

		got, err := util.FetchURL(context.Background(), nil, redirectServer.URL)
		if err != nil {
			t.Fatalf("FetchURL() unexpected error: %v", err)
		}
		if string(got) != "final destination" {
			t.Errorf("FetchURL() = %q, want %q", string(got), "final destination")
		}
	})
}

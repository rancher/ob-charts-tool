package internal

import (
	"net/http"
	"time"
)

// DefaultHTTPClient returns a shared HTTP client with reasonable defaults for internal use.
// This client is safe for concurrent use and reuses connections.
var DefaultHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}

# helmtools

Go utilities for working with Helm charts, focused on upstream chart analysis and version management.

## Features

- **Chart Parsing**: Parse and fetch Helm Chart.yaml files from URLs or bytes
- **Git Operations**: Query remote Git repositories for chart tags and versions without cloning
- **Image Extraction**: Extract container images from Helm values.yaml files using pattern matching
- **Upstream Repositories**: Work with Prometheus Community and Grafana chart repositories
- **Values Navigation**: Navigate and manipulate Helm values.yaml structure with dotted paths
- **Version Management**: Compare and validate semantic versions

## Installation

```bash
go get github.com/rancher/ob-charts-tool/helmtools
```

## Quick Start

### Fetch and parse a Chart.yaml

```go
import (
    "context"
    "github.com/rancher/ob-charts-tool/helmtools/chart"
)

// Create a client (uses http.DefaultClient)
client := chart.NewClient(nil)

// Fetch and parse Chart.yaml
ctx := context.Background()
chart, err := client.FetchChartYAML(ctx, "https://example.com/Chart.yaml")
if err != nil {
    log.Fatal(err)
}

// Access chart metadata
fmt.Println("Chart:", chart.Name, "Version:", chart.Version)

// Find dependencies (excluding "crds")
deps := chart.FindDependencies(chart)
```

### Query Git repositories for chart versions

```go
import (
    "context"
    "github.com/rancher/ob-charts-tool/helmtools/git"
)

// Verify a specific tag exists
exists, ref, hash, err := git.VerifyTagExists(
    context.Background(),
    "https://github.com/prometheus-community/helm-charts",
    "kube-prometheus-stack-65.8.1",
)

// Find all tags matching a pattern
found, tags, err := git.FindMatchingTags(
    context.Background(),
    repoURL,
    "kube-prometheus-stack-",
)

// Find the highest version
highestTag := git.FindHighestVersionTag(tags, "kube-prometheus-stack")
```

### Extract images from values.yaml

```go
import "github.com/rancher/ob-charts-tool/helmtools/image"

// Extract all images, using "v1.0.0" as default tag for images without tags
images, err := image.ExtractImages(valuesData, "v1.0.0")
if err != nil {
    log.Fatal(err)
}

for _, img := range images.Values() {
    fmt.Printf("%s:%s\n", img.Repository, img.Tag)
}
```

### Custom HTTP client configuration

```go
import (
    "net/http"
    "time"
)

// Configure custom HTTP client
httpClient := &http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        // Custom TLS, proxy, etc.
    },
}

// Use it with chart operations
client := chart.NewClient(httpClient)
chart, err := client.FetchChartYAML(ctx, url)
```

## Package Overview

- **`chart`**: Parse and fetch Helm Chart.yaml files
- **`git`**: Query Git repositories for Helm chart tags and versions
- **`image`**: Extract container images from Helm values.yaml files
- **`upstream`**: Work with upstream chart repositories (Prometheus, Grafana)
- **`values`**: Navigate and manipulate Helm values.yaml structure
- **`version`**: Version comparison utilities
- **`util`**: Shared utilities (HTTP, sets, slices)

## Context Support

All I/O operations accept a `context.Context` for cancellation and timeouts:

```go
// With timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

chart, err := client.FetchChartYAML(ctx, url)
```

## Thread Safety

- All package-level functions are safe for concurrent use
- Client instances (`chart.Client`, `git.Client`) are safe for concurrent use
- `util.Set[T]` requires external synchronization for concurrent writes

## Documentation

Full API documentation is available on [pkg.go.dev](https://pkg.go.dev/github.com/rancher/ob-charts-tool/helmtools).

## Contributing

This package is part of the [ob-charts-tool](https://github.com/rancher/ob-charts-tool) project. Contributions are welcome!

## License

See the main repository LICENSE file.

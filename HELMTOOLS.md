# helmtools - Reusable Helm Chart Analysis Library

The `helmtools` packages provide reusable primitives for analyzing Helm charts, extracting images, navigating YAML structures, and working with upstream chart repositories.

## Installation

```bash
go get github.com/rancher/ob-charts-tool/helmtools
```

## Available Packages

### Core Packages
- **`helmtools/chart`** - Chart.yaml parsing and dependency extraction
- **`helmtools/values`** - YAML path navigation and subchart rules
- **`helmtools/image`** - Image extraction from values.yaml and templates
- **`helmtools/version`** - Version tag matching with appCo conventions
- **`helmtools/upstream`** - Upstream repository identification and URL building
- **`helmtools/git`** - Git tag verification and operations
- **`helmtools/util`** - Generic utilities (Set, FilterSlice, HTTP)

---

## Usage Examples

### Example 1: Fetch and Analyze Chart from Web

This example shows how to fetch a Helm chart from GitHub and analyze it.

```go
package main

import (
	"fmt"
	"log"

	"github.com/rancher/ob-charts-tool/helmtools/chart"
	"github.com/rancher/ob-charts-tool/helmtools/image"
	"github.com/rancher/ob-charts-tool/helmtools/upstream"
	"github.com/rancher/ob-charts-tool/helmtools/util"
)

func main() {
	// 1. Identify the upstream repository
	chartName := "kube-state-metrics"
	repo := upstream.IdentifyRepository(chartName)
	fmt.Printf("Repository: %s\n", repo)

	// 2. Build URL for Chart.yaml
	commitHash := "a50b5d92eb96d9e8b234e039f8a98b6e7e84bcf5" // example commit
	chartURL := upstream.BuildChartYAMLURL(chartName, commitHash)
	fmt.Printf("Fetching from: %s\n", chartURL)

	// 3. Fetch and parse Chart.yaml
	c, err := chart.FetchChartYAML(chartURL)
	if err != nil {
		log.Fatalf("Failed to fetch chart: %v", err)
	}

	fmt.Printf("Chart: %s\n", chartName)
	fmt.Printf("Version: %s\n", c.Version)
	fmt.Printf("AppVersion: %s\n", c.AppVersion)

	// 4. Extract dependencies
	deps := chart.FindDependencies(c)
	fmt.Printf("Dependencies (%d):\n", len(deps))
	for _, dep := range deps {
		fmt.Printf("  - %s: %s\n", dep.Name, dep.Version)
	}

	// 5. Fetch values.yaml
	valuesURL := upstream.BuildValuesYAMLURL(chartName, commitHash)
	valuesData, err := util.GetHTTPBody(valuesURL)
	if err != nil {
		log.Fatalf("Failed to fetch values: %v", err)
	}

	// 6. Extract images from values.yaml
	images, err := image.ExtractImages(valuesData, c.AppVersion)
	if err != nil {
		log.Fatalf("Failed to extract images: %v", err)
	}

	fmt.Printf("\nImages found (%d):\n", images.Len())
	for img := range images.ValuesChan() {
		fmt.Printf("  - %s/%s:%s\n", img.Registry, img.Repository, img.Tag)
	}
}
```

**Output:**
```
Repository: https://github.com/prometheus-community/helm-charts.git
Fetching from: https://github.com/prometheus-community/helm-charts/raw/a50b5d92eb96d9e8b234e039f8a98b6e7e84bcf5/charts/kube-state-metrics/Chart.yaml
Chart: kube-state-metrics
Version: 5.15.0
AppVersion: 2.10.0
Dependencies (0):

Images found (2):
  - registry.k8s.io/kube-state-metrics/kube-state-metrics:v2.10.0
  - registry.k8s.io/kube-rbac-proxy/kube-rbac-proxy:v0.15.0
```

---

### Example 2: Analyze Local Chart Files

This example shows how to work with chart files from a cloned git repository.

```go
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/rancher/ob-charts-tool/helmtools/chart"
	"github.com/rancher/ob-charts-tool/helmtools/image"
	"github.com/rancher/ob-charts-tool/helmtools/values"
	"github.com/rancher/ob-charts-tool/helmtools/version"
	"gopkg.in/yaml.v3"
)

func main() {
	// Path to your cloned chart repository
	chartPath := "/path/to/helm-charts/charts/kube-prometheus-stack"

	// 1. Read and parse local Chart.yaml
	chartYAML, err := os.ReadFile(filepath.Join(chartPath, "Chart.yaml"))
	if err != nil {
		log.Fatalf("Failed to read Chart.yaml: %v", err)
	}

	c, err := chart.ParseChartYAML(chartYAML)
	if err != nil {
		log.Fatalf("Failed to parse Chart.yaml: %v", err)
	}

	fmt.Printf("Chart: kube-prometheus-stack\n")
	fmt.Printf("Version: %s\n", c.Version)
	fmt.Printf("AppVersion: %s\n", c.AppVersion)

	// 2. Extract dependencies
	deps := chart.FindDependencies(c)
	fmt.Printf("\nDependencies (%d):\n", len(deps))
	for _, dep := range deps {
		fmt.Printf("  - %s: %s (from: %s)\n", dep.Name, dep.Version, dep.Repository)
	}

	// 3. Read and parse local values.yaml
	valuesYAML, err := os.ReadFile(filepath.Join(chartPath, "values.yaml"))
	if err != nil {
		log.Fatalf("Failed to read values.yaml: %v", err)
	}

	var valuesData map[string]interface{}
	if err := yaml.Unmarshal(valuesYAML, &valuesData); err != nil {
		log.Fatalf("Failed to parse values.yaml: %v", err)
	}

	// 4. Navigate to specific values
	grafanaTag, found := values.NavigatePath(valuesData, "grafana.image.tag")
	if found {
		fmt.Printf("\nGrafana image tag: %s\n", grafanaTag)
	}

	// 5. Extract all images from values.yaml
	images, err := image.ExtractImages(valuesYAML, c.AppVersion)
	if err != nil {
		log.Fatalf("Failed to extract images: %v", err)
	}

	fmt.Printf("\nImages found in values.yaml (%d):\n", images.Len())
	for img := range images.ValuesChan() {
		if img.Registry != "" {
			fmt.Printf("  - %s/%s:%s\n", img.Registry, img.Repository, img.Tag)
		} else {
			fmt.Printf("  - %s:%s\n", img.Repository, img.Tag)
		}
	}

	// 6. Analyze subchart values (example: grafana)
	grafanaPath := filepath.Join(chartPath, "charts", "grafana")
	if _, err := os.Stat(grafanaPath); err == nil {
		analyzeSubchart(grafanaPath, "grafana")
	}
}

func analyzeSubchart(chartPath string, chartName string) {
	fmt.Printf("\n--- Analyzing subchart: %s ---\n", chartName)

	// Read subchart Chart.yaml
	chartYAML, err := os.ReadFile(filepath.Join(chartPath, "Chart.yaml"))
	if err != nil {
		log.Printf("Failed to read %s Chart.yaml: %v", chartName, err)
		return
	}

	c, err := chart.ParseChartYAML(chartYAML)
	if err != nil {
		log.Printf("Failed to parse %s Chart.yaml: %v", chartName, err)
		return
	}

	fmt.Printf("Version: %s, AppVersion: %s\n", c.Version, c.AppVersion)

	// Read subchart values.yaml
	valuesYAML, err := os.ReadFile(filepath.Join(chartPath, "values.yaml"))
	if err != nil {
		log.Printf("Failed to read %s values.yaml: %v", chartName, err)
		return
	}

	var valuesData map[string]interface{}
	if err := yaml.Unmarshal(valuesYAML, &valuesData); err != nil {
		log.Printf("Failed to parse %s values.yaml: %v", chartName, err)
		return
	}

	// Check if image tag matches appVersion
	imageTag, found := values.NavigatePath(valuesData, "image.tag")
	if found {
		matches := version.TagMatchesExpected(imageTag, c.AppVersion)
		fmt.Printf("Image tag: %s (matches appVersion: %t)\n", imageTag, matches)
	}
}
```

**Output:**
```
Chart: kube-prometheus-stack
Version: 55.5.0
AppVersion: v0.70.0

Dependencies (9):
  - kube-state-metrics: 5.15.* (from: https://prometheus-community.github.io/helm-charts)
  - prometheus-node-exporter: 4.24.* (from: https://prometheus-community.github.io/helm-charts)
  - grafana: 7.0.* (from: https://grafana.github.io/helm-charts)
  ...

Grafana image tag: 10.2.2

Images found in values.yaml (15):
  - quay.io/prometheus/prometheus:v2.48.0
  - quay.io/prometheus/alertmanager:v0.26.0
  - quay.io/prometheus-operator/prometheus-operator:v0.70.0
  ...

--- Analyzing subchart: grafana ---
Version: 7.0.8, AppVersion: 10.2.2
Image tag: 10.2.2 (matches appVersion: true)
```

---

### Example 3: Verify Git Tags for Chart Releases

This example shows how to verify that a chart version tag exists in the upstream repository.

```go
package main

import (
	"fmt"
	"log"

	"github.com/rancher/ob-charts-tool/helmtools/git"
	"github.com/rancher/ob-charts-tool/helmtools/upstream"
)

func main() {
	chartName := "kube-prometheus-stack"
	version := "55.5.0"

	// 1. Get repository URL
	repo := upstream.IdentifyRepository(chartName)

	// 2. Build expected tag name
	expectedTag := fmt.Sprintf("%s-%s", chartName, version)

	// 3. Verify tag exists
	exists, tagRef, commitHash, err := git.VerifyTagExists(string(repo), expectedTag)
	if err != nil {
		log.Fatalf("Failed to verify tag: %v", err)
	}

	if exists {
		fmt.Printf("✓ Tag found: %s\n", expectedTag)
		fmt.Printf("  Ref: %s\n", tagRef)
		fmt.Printf("  Commit: %s\n", commitHash)
	} else {
		fmt.Printf("✗ Tag not found: %s\n", expectedTag)
	}

	// 4. Find all tags matching a pattern
	found, tags, err := git.FindMatchingTags(string(repo), chartName+"-55")
	if err != nil {
		log.Fatalf("Failed to find tags: %v", err)
	}

	if found {
		fmt.Printf("\nFound %d tags matching '%s-55':\n", len(tags), chartName)
		for _, tag := range tags {
			fmt.Printf("  - %s (%s)\n", tag.Name, tag.CommitHash[:8])
		}
	}

	// 5. Find highest version tag
	highestTag := git.FindHighestVersionTag(tags, chartName)
	if highestTag != nil {
		fmt.Printf("\nHighest version: %s\n", highestTag.Name)
	}
}
```

**Output:**
```
✓ Tag found: kube-prometheus-stack-55.5.0
  Ref: refs/tags/kube-prometheus-stack-55.5.0
  Commit: a50b5d92eb96d9e8b234e039f8a98b6e7e84bcf5

Found 6 tags matching 'kube-prometheus-stack-55':
  - kube-prometheus-stack-55.0.0 (1a2b3c4d)
  - kube-prometheus-stack-55.1.0 (2b3c4d5e)
  - kube-prometheus-stack-55.2.0 (3c4d5e6f)
  - kube-prometheus-stack-55.3.0 (4d5e6f7g)
  - kube-prometheus-stack-55.4.0 (5e6f7g8h)
  - kube-prometheus-stack-55.5.0 (a50b5d92)

Highest version: kube-prometheus-stack-55.5.0
```

---

### Example 4: Custom Subchart Version Validation

This example shows how to define and use custom rules for validating subchart image tags.

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rancher/ob-charts-tool/helmtools/chart"
	"github.com/rancher/ob-charts-tool/helmtools/values"
	"github.com/rancher/ob-charts-tool/helmtools/version"
	"gopkg.in/yaml.v3"
)

func main() {
	// Define custom rules for your subcharts
	customRules := map[string][]values.SubchartRule{
		"kube-state-metrics": {
			// kube-state-metrics expects "v" prefix on image tag
			{
				ValuesKey: "image.tag",
				PrepareFunc: func(appVersion string) string {
					return "v" + appVersion
				},
			},
		},
		"node-exporter": {
			// node-exporter uses tag as-is
			{ValuesKey: "image.tag"},
		},
	}

	defaultRules := []values.SubchartRule{
		{ValuesKey: "image.tag"},
	}

	// Path to subchart
	subchartPath := "/path/to/charts/kube-prometheus-stack/charts/kube-state-metrics"

	// 1. Read Chart.yaml
	chartYAML, _ := os.ReadFile(filepath.Join(subchartPath, "Chart.yaml"))
	c, _ := chart.ParseChartYAML(chartYAML)

	// 2. Read values.yaml
	valuesYAML, _ := os.ReadFile(filepath.Join(subchartPath, "values.yaml"))
	var valuesData map[string]interface{}
	yaml.Unmarshal(valuesYAML, &valuesData)

	// 3. Get rules for this subchart
	subchartName := values.NormalizeName("rancher-kube-state-metrics")
	rules := values.GetRules(subchartName, customRules, defaultRules)

	// 4. Check if tags match expected values
	mismatches := version.CheckTagsInValues(rules, c.AppVersion, valuesData)

	if len(mismatches) == 0 {
		fmt.Printf("✓ All image tags match appVersion (%s)\n", c.AppVersion)
	} else {
		fmt.Printf("✗ Found %d mismatches:\n", len(mismatches))
		for _, m := range mismatches {
			fmt.Printf("  - %s: expected '%s', got '%s'\n",
				m.ValuesKey, m.ExpectedValue, m.ActualValue)
		}
	}

	// 5. Demonstrate version matching flexibility
	testCases := []struct {
		actual   string
		expected string
	}{
		{"v2.10.0", "v2.10.0"},       // Exact match
		{"2.10.0", "v2.10.0"},         // Missing "v" prefix
		{"v2.10.0-1", "v2.10.0"},      // Build revision suffix
		{"2.10.0-10.11", "v2.10.0"},   // appCo style suffix
	}

	fmt.Println("\nVersion matching examples:")
	for _, tc := range testCases {
		matches := version.TagMatchesExpected(tc.actual, tc.expected)
		fmt.Printf("  %s vs %s: %t\n", tc.actual, tc.expected, matches)
	}
}
```

**Output:**
```
✓ All image tags match appVersion (2.10.0)

Version matching examples:
  v2.10.0 vs v2.10.0: true
  2.10.0 vs v2.10.0: true
  v2.10.0-1 vs v2.10.0: true
  2.10.0-10.11 vs v2.10.0: true
```

---

### Example 5: Extract Images from Rendered Templates

This example shows how to extract images from `helm template` output.

```go
package main

import (
	"fmt"
	"log"
	"os/exec"

	"github.com/rancher/ob-charts-tool/helmtools/image"
)

func main() {
	// 1. Render chart templates using helm CLI
	cmd := exec.Command("helm", "template", "my-release",
		"/path/to/chart",
		"--values", "/path/to/custom-values.yaml")

	output, err := cmd.Output()
	if err != nil {
		log.Fatalf("Failed to render templates: %v", err)
	}

	// 2. Extract images from rendered YAML
	images := image.ExtractImagesFromTemplates(string(output))

	fmt.Printf("Images found in rendered templates (%d):\n", images.Len())

	// 3. Categorize images
	rancherImages := make([]string, 0)
	otherImages := make([]string, 0)
	templateVarImages := make([]string, 0)

	for img := range images.ValuesChan() {
		if contains(img, "{{") {
			templateVarImages = append(templateVarImages, img)
		} else if contains(img, "rancher/") {
			rancherImages = append(rancherImages, img)
		} else {
			otherImages = append(otherImages, img)
		}
	}

	if len(rancherImages) > 0 {
		fmt.Println("\nRancher images:")
		for _, img := range rancherImages {
			fmt.Printf("  - %s\n", img)
		}
	}

	if len(otherImages) > 0 {
		fmt.Println("\nOther images:")
		for _, img := range otherImages {
			fmt.Printf("  - %s\n", img)
		}
	}

	if len(templateVarImages) > 0 {
		fmt.Println("\nImages with template variables (need manual check):")
		for _, img := range templateVarImages {
			fmt.Printf("  - %s\n", img)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && 
		len(s) >= len(substr) && 
		findIndex(s, substr) >= 0
}

func findIndex(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
```

**Output:**
```
Images found in rendered templates (12):

Rancher images:
  - rancher/mirrored-prometheus-operator:v0.70.0
  - rancher/mirrored-prometheus:v2.48.0
  - rancher/mirrored-alertmanager:v0.26.0

Other images:
  - quay.io/prometheus/node-exporter:v1.7.0
  - grafana/grafana:10.2.2

Images with template variables (need manual check):
  - {{ .Values.image.repository }}:{{ .Values.image.tag }}
```

---

## API Reference

### helmtools/chart

```go
// ParseChartYAML parses Chart.yaml bytes into a Chart struct
func ParseChartYAML(data []byte) (*Chart, error)

// FetchChartYAML fetches Chart.yaml from a URL and parses it
func FetchChartYAML(url string) (*Chart, error)

// FindDependencies extracts dependencies, filtering out "crds"
func FindDependencies(chart *Chart) []ChartDependency
```

### helmtools/values

```go
// NavigatePath follows a dotted key path through a YAML map
func NavigatePath(data map[string]interface{}, keyPath string) (string, bool)

// NavigateMap follows a dotted key path and returns the nested map
func NavigateMap(data map[string]interface{}, keyPath string) (map[string]interface{}, bool)

// GetRules returns applicable rules for a subchart
func GetRules(name string, ruleMap map[string][]SubchartRule, defaults []SubchartRule) []SubchartRule
```

### helmtools/version

```go
// TagMatchesExpected checks version tag compatibility (handles "v" prefix and build suffixes)
func TagMatchesExpected(actual, expected string) bool

// CheckTagsInValues validates image tags against rules
func CheckTagsInValues(rules []SubchartRule, appVersion string, valuesMap map[string]interface{}) []TagMismatch
```

### helmtools/image

```go
// ExtractImages recursively extracts images from values.yaml
func ExtractImages(valuesData []byte, defaultTag string) (util.Set[Image], error)

// ExtractImagesFromTemplates extracts images from helm template output
func ExtractImagesFromTemplates(renderedChart string) util.Set[string]
```

### helmtools/upstream

```go
// IdentifyRepository determines which repository a chart belongs to
func IdentifyRepository(chartName string) Repository

// BuildChartYAMLURL builds the raw GitHub URL for Chart.yaml
func BuildChartYAMLURL(chartName string, commitHash string) string

// BuildValuesYAMLURL builds the raw GitHub URL for values.yaml
func BuildValuesYAMLURL(chartName string, commitHash string) string
```

### helmtools/git

```go
// VerifyTagExists checks if a tag exists in a remote repository
func VerifyTagExists(repoURL string, tag string) (bool, string, string, error)

// FindMatchingTags finds all tags matching a pattern
func FindMatchingTags(repoURL string, tagPartial string) (bool, []Tag, error)

// FindHighestVersionTag finds the highest semantic version tag
func FindHighestVersionTag(tags []Tag, componentPrefix string) *Tag
```

### helmtools/util

```go
// Set[T] - Generic set implementation
type Set[T comparable] interface {
    Add(T) bool
    Contains(T) bool
    Remove(T) bool
    Values() []T
    Len() int
    // ... more methods
}

// FilterSlice filters a slice based on a predicate
func FilterSlice[T any](slice []T, fn func(T) bool) []T

// GetHTTPBody fetches a URL and returns the response body
func GetHTTPBody(url string) ([]byte, error)
```

---

## Best Practices

### 1. Error Handling
Always check errors when fetching remote resources:
```go
c, err := chart.FetchChartYAML(url)
if err != nil {
    log.Fatalf("Failed to fetch chart: %v", err)
}
```

### 2. Version Matching
Use `version.TagMatchesExpected()` for flexible version comparison:
```go
// Handles "v" prefix differences and build suffixes
matches := version.TagMatchesExpected("2.10.0-1", "v2.10.0") // true
```

### 3. Custom Rules
Define rules for your specific use case:
```go
rules := map[string][]values.SubchartRule{
    "my-chart": {
        {
            ValuesKey: "image.tag",
            PrepareFunc: func(v string) string { return "v" + v },
        },
    },
}
```

### 4. Working with Sets
The `util.Set` type provides efficient deduplication:
```go
images, _ := image.ExtractImages(valuesData, appVersion)
for img := range images.ValuesChan() {
    // Process unique images
}
```

---

## Common Patterns

### Pattern: Analyze Chart and All Subcharts
```go
func analyzeChartTree(chartPath string) {
    // Parse main chart
    c, _ := parseLocalChart(chartPath)
    
    // Analyze dependencies
    deps := chart.FindDependencies(c)
    for _, dep := range deps {
        subchartPath := filepath.Join(chartPath, "charts", dep.Name)
        if exists(subchartPath) {
            analyzeChartTree(subchartPath) // Recursive
        }
    }
}
```

### Pattern: Validate Chart Before Release
```go
func validateChart(chartPath string) []error {
    errors := []error{}
    
    // 1. Parse chart
    c, err := parseLocalChart(chartPath)
    if err != nil {
        errors = append(errors, err)
        return errors
    }
    
    // 2. Check version format
    if _, err := semver.NewVersion(c.Version); err != nil {
        errors = append(errors, fmt.Errorf("invalid version: %w", err))
    }
    
    // 3. Validate image tags match appVersion
    // ... validation logic
    
    return errors
}
```

---

## Contributing

These packages are extracted from the `rancher/ob-charts-tool` CLI. To contribute:

1. Keep packages generic and reusable
2. Add tests for new functionality
3. Update this documentation with examples
4. Ensure no CLI-specific dependencies leak into helmtools

## License

See the main repository LICENSE file.

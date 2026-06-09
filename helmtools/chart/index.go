package chart

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// Index represents a Helm chart repository index.yaml file.
// See https://helm.sh/docs/topics/chart_repository/#the-index-file
type Index struct {
	APIVersion string                  `yaml:"apiVersion"`
	Entries    map[string][]IndexEntry `yaml:"entries"`
	Generated  time.Time               `yaml:"generated"`
}

// IndexEntry represents a single chart version in a Helm repository index.
type IndexEntry struct {
	Name        string            `yaml:"name"`
	Version     string            `yaml:"version"`
	Description string            `yaml:"description"`
	APIVersion  string            `yaml:"apiVersion"`
	AppVersion  string            `yaml:"appVersion"`
	Type        string            `yaml:"type"`
	URLs        []string          `yaml:"urls"`
	Created     time.Time         `yaml:"created"`
	Digest      string            `yaml:"digest"`
	Keywords    []string          `yaml:"keywords"`
	Maintainers []Maintainer      `yaml:"maintainers"`
	Home        string            `yaml:"home"`
	Sources     []string          `yaml:"sources"`
	Icon        string            `yaml:"icon"`
	Deprecated  bool              `yaml:"deprecated"`
	Annotations map[string]string `yaml:"annotations"`
}

// Maintainer represents a chart maintainer.
type Maintainer struct {
	Name  string `yaml:"name"`
	Email string `yaml:"email"`
	URL   string `yaml:"url"`
}

// ParseIndex parses a Helm repository index.yaml file.
func ParseIndex(data []byte) (*Index, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("index data cannot be empty")
	}

	var index Index
	if err := yaml.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to parse index.yaml: %w", err)
	}

	return &index, nil
}

// LatestVersion returns the latest version of a chart from the index.
// Returns nil if the chart is not found or has no versions.
// The index entries are assumed to be sorted with the latest first.
func (idx *Index) LatestVersion(chartName string) *IndexEntry {
	entries, exists := idx.Entries[chartName]
	if !exists || len(entries) == 0 {
		return nil
	}
	return &entries[0]
}

// ChartVersions returns all versions of a chart from the index.
// Returns an empty slice if the chart is not found.
func (idx *Index) ChartVersions(chartName string) []IndexEntry {
	entries, exists := idx.Entries[chartName]
	if !exists {
		return []IndexEntry{}
	}
	return entries
}

// ListCharts returns all chart names in the index.
func (idx *Index) ListCharts() []string {
	charts := make([]string, 0, len(idx.Entries))
	for name := range idx.Entries {
		charts = append(charts, name)
	}
	return charts
}

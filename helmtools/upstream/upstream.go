package upstream

import (
	"fmt"
	"strings"
)

const (
	grafanaRawURL       = "https://github.com/grafana-community/helm-charts/raw/%s/charts/%s/%s.yaml"
	promCommunityRawURL = "https://github.com/prometheus-community/helm-charts/raw/%s/charts/%s/%s.yaml"
)

// IdentifyRepository determines which upstream repository a chart belongs to.
// Currently supports Grafana and Prometheus Community charts.
func IdentifyRepository(chartName string) Repository {
	if strings.Contains(chartName, "grafana") {
		return RepositoryGrafana
	}
	return RepositoryPrometheus
}

// BuildChartYAMLURL builds the raw GitHub URL for a chart's Chart.yaml file.
func BuildChartYAMLURL(chartName string, commitHash string) string {
	repo := IdentifyRepository(chartName)
	switch repo {
	case RepositoryGrafana:
		return fmt.Sprintf(grafanaRawURL, commitHash, chartName, "Chart")
	case RepositoryPrometheus:
		return fmt.Sprintf(promCommunityRawURL, commitHash, chartName, "Chart")
	default:
		return ""
	}
}

// BuildValuesYAMLURL builds the raw GitHub URL for a chart's values.yaml file.
func BuildValuesYAMLURL(chartName string, commitHash string) string {
	repo := IdentifyRepository(chartName)
	switch repo {
	case RepositoryGrafana:
		return fmt.Sprintf(grafanaRawURL, commitHash, chartName, "values")
	case RepositoryPrometheus:
		return fmt.Sprintf(promCommunityRawURL, commitHash, chartName, "values")
	default:
		return ""
	}
}

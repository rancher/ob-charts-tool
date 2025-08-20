package upstream

import (
	"fmt"
	"strings"

	gitremote "github.com/rancher/ob-charts-tool/internal/git/remote"
	log "github.com/sirupsen/logrus"
)

const (
	grafanaChartsURL    = "https://github.com/grafana/helm-charts.git"
	prometheusChartsURL = "https://github.com/prometheus-community/helm-charts.git"
	versionTemplate     = "kube-prometheus-stack-%s"
	promCommunityRawURL = "https://github.com/prometheus-community/helm-charts/raw/%s/charts/%s/%s.yaml"
	grafanaRawURL       = "https://github.com/grafana/helm-charts/raw/%s/charts/%s/%s.yaml"
)

func IdentifyChartUpstream(chartName string) string {
	if strings.Contains(chartName, "grafana") {
		return grafanaChartsURL
	}

	return prometheusChartsURL
}

func GetKubePrometheusVersionTag(versionNumber string) string {
	return fmt.Sprintf(versionTemplate, versionNumber)
}

func PrometheusChartVersionExists(version string) (bool, string, string) {
	return gitremote.VerifyTagExists(prometheusChartsURL, GetKubePrometheusVersionTag(version))
}

func GetChartValuesURL(chartName string, chartHash string) string {
	url := ""
	switch chartName {
	case "grafana":
		url = fmt.Sprintf(grafanaRawURL, chartHash, chartName, "values")
	default:
		url = fmt.Sprintf(promCommunityRawURL, chartHash, chartName, "values")
	}

	if url != "" {
		return url
	}

	log.Errorf("Cannot find values file URL for `%s`", chartName)
	log.Exit(1)
	return ""
}

func GetChartsChartURL(chartName string, chartHash string) string {
	url := ""
	switch chartName {
	case "grafana":
		url = fmt.Sprintf(grafanaRawURL, chartHash, chartName, "Chart")
	default:
		url = fmt.Sprintf(promCommunityRawURL, chartHash, chartName, "Chart")
	}

	if url != "" {
		return url
	}

	log.Errorf("Cannot find chart.yaml file URL for `%s`", chartName)
	log.Exit(1)
	return ""
}

package upstream

import (
	"fmt"
	"github.com/mallardduck/ob-charts-tool/internal/git"
	log "github.com/sirupsen/logrus"
	"strings"
)

const (
	grafanaChartsURL       = "https://github.com/grafana/helm-charts.git"
	prometheusChartsURL    = "https://github.com/prometheus-community/helm-charts.git"
	versionTemplate        = "kube-prometheus-stack-%s"
	promCommunityValuesURL = "https://github.com/prometheus-community/helm-charts/raw/%s/charts/%s/values.yaml"
	grafanaValuesURL       = "https://github.com/grafana/helm-charts/raw/%s/charts/grafana/values.yaml"
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
	return git.VerifyTagExists(prometheusChartsURL, GetKubePrometheusVersionTag(version))
}

func GetChartValuesURL(chartName string, chartHash string) string {
	url := ""
	switch chartName {
	case "grafana":
		url = fmt.Sprintf(grafanaValuesURL, chartHash)
	default:
		url = fmt.Sprintf(promCommunityValuesURL, chartHash, chartName)
	}

	if url != "" {
		return url
	}

	log.Errorf("Cannot find values file URL for `%s`", chartName)
	log.Exit(1)
	return ""
}

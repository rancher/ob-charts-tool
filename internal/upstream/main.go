package upstream

import (
	"fmt"
	"github.com/mallardduck/ob-charts-tool/internal/git"
	"strings"
)

const (
	grafanaChartsURL    = "https://github.com/grafana/helm-charts.git"
	prometheusChartsURL = "https://github.com/prometheus-community/helm-charts.git"
	versionTemplate     = "kube-prometheus-stack-%s"
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

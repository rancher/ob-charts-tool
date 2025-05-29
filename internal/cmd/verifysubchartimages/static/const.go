package static

import (
	"errors"
	"os"
	"path/filepath"
)

type MonitoringSubChartName string

const (
	GrafanaSubChart                  MonitoringSubChartName = "grafana"
	KubeStateMetricsSubChart         MonitoringSubChartName = "kube-state-metrics"
	NodeExporterSubChart             MonitoringSubChartName = "node-exporter"
	PrometheusAdapterSubChart        MonitoringSubChartName = "prometheus-adapter"
	PushProxSubChart                 MonitoringSubChartName = "pushprox"
	WindowsExporterSubChart          MonitoringSubChartName = "windows-exporter"
	ProjectMonitoringGrafanaSubChart MonitoringSubChartName = "project-monitoring-grafana" // This is a sub-sub-chart...meaning it depends on Grafana being done first
)

func (n MonitoringSubChartName) LegacyName() MonitoringSubChartName {
	return "rancher-" + n
}

func (n MonitoringSubChartName) String() string {
	return string(n)
}

func (n MonitoringSubChartName) IdentifyLocalPath(workingPath string) (string, error) {
	subChartPath := filepath.Join(workingPath, n.String())
	if _, err := os.Stat(subChartPath); os.IsNotExist(err) {
		subChartPath = filepath.Join(workingPath, n.LegacyName().String())
		if _, err := os.Stat(subChartPath); os.IsNotExist(err) {
			return "", errors.New("cannot identify correct local path for this subchart")
		}

		return subChartPath, nil
	}

	return subChartPath, nil
}

// MonitoringSubCharts lists all the subcharts for `rancher-monitoring`
func MonitoringSubCharts() []MonitoringSubChartName {
	return []MonitoringSubChartName{
		GrafanaSubChart,
		KubeStateMetricsSubChart,
		NodeExporterSubChart,
		PrometheusAdapterSubChart,
		PushProxSubChart,
		WindowsExporterSubChart,
	}
}

// MonitoringSubChartsWithPrefix lists all the subcharts for `rancher-monitoring` including the rancher prefix
// Notably, this prefixing is something we don't need to do and should eliminate so that's why we provide both.
func MonitoringSubChartsWithPrefix() []MonitoringSubChartName {
	return []MonitoringSubChartName{
		GrafanaSubChart.LegacyName(),
		KubeStateMetricsSubChart.LegacyName(),
		NodeExporterSubChart.LegacyName(),
		PrometheusAdapterSubChart.LegacyName(),
		PushProxSubChart.LegacyName(),
		WindowsExporterSubChart.LegacyName(),
	}
}

// AppVersionRule represents a static rule of what values.yaml file and field should be synced with .Chart.AppVersion
type AppVersionRule struct {
	ValuesKey   string
	PrepareFunc func(string) string
}

var AppVersionEnabled = map[MonitoringSubChartName]bool{
	GrafanaSubChart:           true,
	KubeStateMetricsSubChart:  true,
	"rancher-monitoring":      false,
	NodeExporterSubChart:      true,
	PrometheusAdapterSubChart: true,
	WindowsExporterSubChart:   true,
}

type AppVersionRuleList []AppVersionRule

type AppVersionRulesMap map[MonitoringSubChartName]AppVersionRuleList

var AppVersionRules AppVersionRulesMap = AppVersionRulesMap{
	KubeStateMetricsSubChart: {
		{
			ValuesKey:   ".Values.image.tag",
			PrepareFunc: PrefixWithV,
		},
		{
			ValuesKey:   ".Values.kubeRBACProxy.image.tag",
			PrepareFunc: PrefixWithV,
		},
	},
}

func (m AppVersionRulesMap) SubChartHasRules(subChart MonitoringSubChartName) bool {
	_, ok := m[subChart]
	return ok
}

func (m AppVersionRulesMap) GetSubChartRules(subChart MonitoringSubChartName) AppVersionRuleList {
	return m[subChart]
}

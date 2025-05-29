package static

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

func (n MonitoringSubChartName) String() string {
	return string(n)
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
		"rancher-" + GrafanaSubChart,
		"rancher-" + KubeStateMetricsSubChart,
		"rancher-" + NodeExporterSubChart,
		"rancher-" + PrometheusAdapterSubChart,
		"rancher-" + PushProxSubChart,
		"rancher-" + WindowsExporterSubChart,
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

var AppVersionRules = map[MonitoringSubChartName][]AppVersionRule{
	KubeStateMetricsSubChart: []AppVersionRule{
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

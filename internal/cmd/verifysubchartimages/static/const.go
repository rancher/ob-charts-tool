package static

// AppVersionRule represents a static rule of what values.yaml file and field should be synced with .Chart.AppVersion
type AppVersionRule struct {
	ValuesKey   string
	PrepareFunc func(string) string
}

var AppVersionEnabled = map[string]bool{
	"rancher-prometheus-adapter": true,
	"rancher-kube-state-metrics": true,
	"rancher-node-exporter":      true,
	"rancher-monitoring":         false,
	"rancher-grafana":            true,
	"rancher-windows-exporter":   true,
}

func PrefixWithV(in string) string {
	return "v" + in
}

var AppVersionRules = map[string][]AppVersionRule{
	"rancher-kube-state-metrics": []AppVersionRule{
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

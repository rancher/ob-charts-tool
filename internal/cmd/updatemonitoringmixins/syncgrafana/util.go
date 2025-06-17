package syncgrafana

import "github.com/rancher/ob-charts-tool/internal/cmd/updatemonitoringmixins/types"

func setDefaultMaxK8s(ds types.DashboardSource) types.DashboardSource {
	if ds.GetMaxKubernetes() == "" {
		// Equal to: https://github.com/prometheus-community/helm-charts/blob/0b60795bb66a21cd368b657f0665d67de3e49da9/charts/kube-prometheus-stack/hack/sync_grafana_dashboards.py#L326
		ds.SetMaxKubernetes("9.9.9-9")
	}

	return ds
}

package charts

import (
	"fmt"
)

func BaseMonitoringDir(cwd string) string {
	return fmt.Sprintf("%s/charts/rancher-monitoring", cwd)
}

func BaseMonitoringVersionDir(cwd string, version string) string {
	return fmt.Sprintf("%s/charts/rancher-monitoring/%s", cwd, version)
}

package syncprom

import (
	"fmt"
	"github.com/rancher/ob-charts-tool/internal/cmd/updatemonitoringmixins/config"
	"github.com/rancher/ob-charts-tool/internal/cmd/updatemonitoringmixins/git"
)

func SyncPrometheusRules(tempDir string, repos []git.RepoConfigStatus) error {
	chartPath := config.GetContext().ChartRootDir
	fmt.Println(chartPath)

	// repoSHAs is equivalent to `refs` from `sync_grafana_dashboards.py` or `sync_prometheus_rules.py`
	repoSHAs := git.RepoSHAs(repos)
	fmt.Println(repoSHAs)

	return nil
}

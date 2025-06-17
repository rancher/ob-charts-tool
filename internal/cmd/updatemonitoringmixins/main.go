package updatemonitoringmixins

import (
	"fmt"
	"github.com/jedib0t/go-pretty/text"
	"github.com/rancher/ob-charts-tool/internal/charts"
	"github.com/rancher/ob-charts-tool/internal/cmd/updatemonitoringmixins/constants"
	mixinGit "github.com/rancher/ob-charts-tool/internal/cmd/updatemonitoringmixins/git"
	"github.com/rancher/ob-charts-tool/internal/cmd/updatemonitoringmixins/syncgrafana"
	"github.com/rancher/ob-charts-tool/internal/cmd/updatemonitoringmixins/syncprom"
	"github.com/rancher/ob-charts-tool/internal/git"
	"github.com/rancher/ob-charts-tool/internal/git/cache"
	"github.com/rancher/ob-charts-tool/internal/util"
	"os"
)

type ChartPathMode int

const (
	BasePathModeUnknown ChartPathMode = iota
	BasePathModeOBTeam
	BasePathModeCWD
)

func PrepareGitCache(useCache bool, cacheDir string) {
	if useCache {
		// Called to set root dir
		cache.SetupGitCacheManager(cacheDir)
	}
}

func DetermineTargetRoot(args []string, pathMode ChartPathMode, workingDir string) string {
	var chartTargetRoot string
	if pathMode == BasePathModeOBTeam {
		targetVersion := args[0]
		fmt.Println(
			text.AlignCenter.Apply(
				text.Color.Sprintf(text.FgBlue, "Looking for `rancher-monitoring` chart with version `%s`...", targetVersion),
				125,
			),
		)
		cwd, err := os.Getwd()
		if err != nil {
			panic(err)
		}

		chartTargetRoot = charts.BaseMonitoringVersionDir(cwd, targetVersion)
	} else if pathMode == BasePathModeCWD {
		chartTargetRoot = workingDir
	}

	return chartTargetRoot
}

func VerifySystemDependencies() error {
	// TODO: verify user has jb command installed (jsonnet-bundler)
	return nil
}

// PrepareTempDir creates the temp dir, but callers must be responsible for `defer os.RemoveAll` of those dirs
func PrepareTempDir() (string, error) {
	tempDir, err := os.MkdirTemp("", "ob-charts-tool-mixins-")
	if err != nil {
		fmt.Println("Error creating temporary directory:", err)
		return "", err
	}

	return tempDir, nil
}

// UpdateMonitoringMixins is essentially equivalent to `update_mixins.sh`
func UpdateMonitoringMixins(useCache bool) error {
	// First prepare a temp dir
	tempDir, err := PrepareTempDir()
	defer os.RemoveAll(tempDir) // Ensure cleanup when the function exits
	if err != nil {
		fmt.Println("Error creating temporary directory:", err)
	}
	fmt.Println(tempDir)

	// Initial config for repos to sync rules/dashboards from
	repos := []mixinGit.RepoConfigStatus{
		{
			Name:    "kube-prometheus",
			RepoURL: "https://github.com/prometheus-operator/kube-prometheus.git",
		},
		{
			Name:    "kubernetes-mixin",
			RepoURL: "https://github.com/kubernetes-monitoring/kubernetes-mixin.git",
		},
		{
			Name:    "etcd",
			RepoURL: "https://github.com/etcd-io/etcd.git",
		},
		// TODO: in the future we'll maybe add a Rancher source
	}

	// When the HeadSHA is empty we'll find the head
	// TODO add a force flag later?
	for key, repoConfig := range constants.Repos {
		if repoConfig.HeadSha != "" {
			continue
		}
		headSha, err := git.FindRepoHeadSha(repoConfig.RepoURL)
		if err != nil {
			panic(err)
		}
		repoConfig.HeadSha = headSha
		constants.Repos[key] = repoConfig
	}

	// Clone git repos
	for _, repoConfig := range repos {
		var cloneErr error
		repoConfigParams := git.RepoConfigParams{Name: repoConfig.Name, URL: repoConfig.RepoURL, Head: repoConfig.HeadHash()}
		if useCache {
			destDir := util.GetRepoCloneDir(tempDir, repoConfig.Name, repoConfig.HeadSha) + "/shallow"
			_, cloneErr = git.CachedShallowCloneRepository(repoConfigParams, destDir)
		} else {
			destDir := util.GetRepoCloneDir(tempDir, repoConfig.Name, repoConfig.HeadSha) + "/shallow"
			_, cloneErr = git.ShallowCloneRepository(repoConfigParams, destDir)
		}
		if cloneErr != nil {
			fmt.Println("Error shallow cloning repository:", cloneErr)
			os.Exit(1)
		}
	}

	dashboardsErr := syncgrafana.DashboardsSync(tempDir, repos)
	if dashboardsErr != nil {
		return dashboardsErr
	}

	rulesErr := syncprom.SyncPrometheusRules(tempDir, repos)
	if rulesErr != nil {
		return rulesErr
	}

	return nil
}

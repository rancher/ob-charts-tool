package cmd

import (
	"fmt"
	"github.com/rancher/ob-charts-tool/internal/cmd/updatemonitoringmixins"
	"github.com/rancher/ob-charts-tool/internal/cmd/updatemonitoringmixins/config"
	"github.com/rancher/ob-charts-tool/internal/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	debugMode    = false
	useCache     = true
	disableCache = false
	cacheDir     string
	workingDir   string
	pathMode     = updatemonitoringmixins.BasePathModeOBTeam
)

var updateMonitoringMixinsCmd = &cobra.Command{
	Use:   "updateMonitoringMixins",
	Short: "Update all the monitoring chart mixins",
	PreRun: func(cmd *cobra.Command, args []string) {
		ctx := &config.AppContext{
			DebugMode: false,
		}
		config.SetContext(ctx)
	},
	Args: func(_ *cobra.Command, args []string) error {

		if len(args) == 0 && workingDir != "" {
			pathMode = updatemonitoringmixins.BasePathModeCWD
		}

		// Check if there's one argument provided
		if len(args) == 1 || workingDir != "" {
			return nil
		}

		return fmt.Errorf("you must provide a target monitoring chart version")
	},
	Run: updateMonitoringMixinsHandler,
}

func init() {
	updateMonitoringMixinsCmd.PersistentFlags().BoolVarP(&disableCache, "disableCache", "C", false, "disable the use of caching")
	if disableCache {
		useCache = false
	}
	maybeCacheDir, err := util.GetCacheDir("ob-charts-tool")
	if err == nil {
		cacheDir = maybeCacheDir
	} else {
		logrus.Warn("Cache dir setup failed, cache will not work.")
		logrus.Warnf("attempted using cached directory: %s", maybeCacheDir)
		useCache = false
	}
	updateMonitoringMixinsCmd.PersistentFlags().StringVarP(&workingDir, "working-dir", "w", "", "Specify the working directory to use")
	updateMonitoringMixinsCmd.PersistentFlags().BoolVarP(&debugMode, "debug", "D", false, "enable debug mode")
	rootCmd.AddCommand(updateMonitoringMixinsCmd)
}

func updateMonitoringMixinsHandler(_ *cobra.Command, args []string) {

	updatemonitoringmixins.PrepareGitCache(useCache, cacheDir)
	chartTargetRoot := updatemonitoringmixins.DetermineTargetRoot(args, pathMode, workingDir)

	ctx := config.GetContext()
	ctx.ChartRootDir = chartTargetRoot
	ctx.DebugMode = debugMode

	err := updatemonitoringmixins.VerifySystemDependencies()
	if err != nil {
		logrus.Fatal(err)
		return
	}

	mixinErr := updatemonitoringmixins.UpdateMonitoringMixins(useCache)
	if mixinErr != nil {
		logrus.Fatal(err)
		return
	}
}

package cmd

import (
	"fmt"
	"github.com/rancher/ob-charts-tool/internal/cmd/verifysubchartimages"
	"github.com/rancher/ob-charts-tool/internal/git"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/text"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var workingPath string

// checkSubchartImages represents the checkSubchartImages command
var checkSubchartImages = &cobra.Command{
	Use:   "checkSubchartImages",
	Short: "Sanity check that values.yaml patches match expected results.",
	Long: `Using either a version as first arg, or helm chart debug output from STDIN, this command will output a list
of the necessary images used in the chart. And then verify those are mirrored by the Rancher Image mirror.`,
	Args: func(_ *cobra.Command, args []string) error {
		// Check if there's one argument provided
		if len(args) == 1 {
			return nil
		}

		return fmt.Errorf("you must provide one arg")
	},
	Run: checkSubchartImagesHandler,
}

func init() {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	checkSubchartImages.Flags().StringVarP(&workingPath, "workingDir", "D", cwd, "The root working directory to `ob-team-charts` [starts as CWD]")
	checkSubchartImages.Flags().BoolP("allow-dirty", "Q", false, "Allow the Git repo dirty check to be skipped...use with caution.")
	checkSubchartImages.Flags().BoolP("dynamic-search", "S", false, "Use a search method instead of static rules. [EXPERIMENTAL]")
	rootCmd.AddCommand(checkSubchartImages)
}

func checkSubchartImagesHandler(cmd *cobra.Command, args []string) {
	var packageTargetRoot string
	targetVersion := args[0]
	fmt.Println(
		text.AlignCenter.Apply(
			text.Color.Sprintf(text.FgBlue, "Looking for `rancher-monitoring` chart with version `%s`...", targetVersion),
			125,
		),
	)

	packageTargetRoot = fmt.Sprintf("%s/packages/rancher-monitoring/%s", workingPath, targetVersion)
	if _, err := os.Stat(packageTargetRoot); os.IsNotExist(err) {
		if strings.Count(targetVersion, ".") == 2 {
			lastPeriod := strings.LastIndex(targetVersion, ".")
			alternativeVersion := targetVersion[:lastPeriod]
			packageTargetRoot = fmt.Sprintf("%s/packages/rancher-monitoring/%s", workingPath, alternativeVersion)
			if _, err := os.Stat(packageTargetRoot); os.IsNotExist(err) {
				panic(fmt.Sprintf("Cannot find a monitoring package of the provided version (%s)", targetVersion))
			} else {
				log.Warnf("Couldn't find package version `%s`, however `%s` does exist and will be used.", targetVersion, alternativeVersion)
			}
			targetVersion = alternativeVersion
		}
	}

	fmt.Println(
		text.Color.Sprintf(text.FgBlue, "The %s chart was found - next this tool will run `helm debug` to get the rendered chart.", targetVersion),
	)

	allowDirty, _ := cmd.Flags().GetBool("allow-dirty")
	if !allowDirty {
		// First verify Git repo is in a clean state (`ob-team-charts` repo path should be workingPath)
		if !git.IsRepoClean(workingPath) {
			log.Errorf("%s is not clean", workingPath)
			log.Exit(1)
			return
		}
	}

	dynamicSearch, _ := cmd.Flags().GetBool("dynamic-search")
	if dynamicSearch {
		verifysubchartimages.VerifySubchartImagesDynamic(workingPath, targetVersion, packageTargetRoot)
	} else {
		verifysubchartimages.VerifySubchartImages(workingPath, targetVersion, packageTargetRoot)
	}
}

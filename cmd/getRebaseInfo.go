package cmd

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/mallardduck/ob-charts-tool/internal/cmd/rebaseinfo"

	"github.com/jedib0t/go-pretty/text"
	"github.com/spf13/cobra"
)

// getRebaseInfoCmd represents the getRebaseInfo command
var getRebaseInfoCmd = &cobra.Command{
	Use:   "getRebaseInfo",
	Short: "Collect the basic information about a potential rebase target version",
	Args: func(_ *cobra.Command, args []string) error {
		// Check if there's one argument provided
		if len(args) == 1 {
			return nil
		}

		return fmt.Errorf("you must provide the target upstream chart version")
	},
	Run: getRebaseInfoHandler,
}

func init() {
	rootCmd.AddCommand(getRebaseInfoCmd)
}

func getRebaseInfoHandler(_ *cobra.Command, args []string) {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	fmt.Println("This command will do a series of web requests to identify information about the chart rebase.")
	targetChartVersion := args[0]
	fmt.Println(
		text.AlignCenter.Apply(
			text.Color.Sprintf(text.FgBlue, "Looking for upstream monitorin chart with version `%s`...", targetChartVersion),
			75,
		),
	)

	// VerifyTagExists will either exit or return the tag reference and hash for a given chart version.
	tagRef, hash := rebaseinfo.VerifyTagExists(targetChartVersion)
	rebaseInfoState := rebaseinfo.CollectInfo(targetChartVersion, tagRef, hash)

	/// Some of these TODOs might be better as new commands, some may live here
	// TODO: Compare the found images for updated patch releases
	// TODO: Compare the found images to those used in existing Rancher chart somehow
	// TODO: Consider adding checks against "rancher/image-mirror" repo?

	log.Debug(rebaseInfoState)
	fmt.Println("Rebase information has been collected and will be saved to `rebase.yaml` file.")

	savedRebaseInfoFilePath := rebaseInfoState.SaveStateToRebaseYaml(cwd)
	fmt.Println(fmt.Sprintf("The rebase information is saved at: %s", savedRebaseInfoFilePath))
}

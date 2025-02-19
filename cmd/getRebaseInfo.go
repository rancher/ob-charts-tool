package cmd

import (
	"fmt"
	"os"

	"github.com/mallardduck/ob-charts-tool/internal/cmd/rebaseInfo"

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
	tagRef, hash := rebaseInfo.VerifyTagExists(targetChartVersion)
	rebaseInfoState := rebaseInfo.CollectInfo(targetChartVersion, tagRef, hash)

	fmt.Println(rebaseInfoState)

	rebaseInfoState.SaveStateToRebaseYaml(cwd)
}

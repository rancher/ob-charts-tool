package cmd

import (
	"fmt"
	"os"

	"github.com/jedib0t/go-pretty/text"
	"github.com/mallardduck/ob-charts-tool/internal/upstream"
	log "github.com/sirupsen/logrus"
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

	exists := false
	var tagRef string
	var hash string
	if exists, tagRef, hash = upstream.ChartVersionExists(targetChartVersion); !exists {
		errorText := fmt.Sprintf("Cannot find upstream chart version `%s`", targetChartVersion)
		fmt.Println(
			text.AlignCenter.Apply(
				text.Color.Sprint(text.FgRed, errorText),
				75,
			),
		)
		log.Error(errorText)
		os.Exit(1)
	}

	rebaseRequest := upstream.PrepareRebaseRequestInfo(targetChartVersion, tagRef, hash)
	rebaseInfoState := upstream.CollectRebaseChartsInfo(rebaseRequest)
	_ = rebaseInfoState.FindChartsContainers()
	fmt.Println(rebaseInfoState)

	rebaseInfoState.SaveStateToRebaseYaml(cwd)
}

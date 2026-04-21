package monitoring

import (
	"fmt"
	"os"

	"github.com/jedib0t/go-pretty/text"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/rancher/ob-charts-tool/cmd/groups"
	"github.com/rancher/ob-charts-tool/internal/cmd/rebaseinfo"
	monsubcharts "github.com/rancher/ob-charts-tool/internal/monitoring"
	"github.com/rancher/ob-charts-tool/internal/rebase"
)

// getRebaseInfoCmd represents the getRebaseInfo command
var getRebaseInfoCmd = &cobra.Command{
	Use:     "getRebaseInfo",
	GroupID: groups.MonitoringGroup.ID,
	Short:   "Collect the basic information about a potential rebase target version",
	Args: func(_ *cobra.Command, args []string) error {
		// Check if there's one argument provided
		if len(args) == 1 {
			return nil
		}

		return fmt.Errorf("you must provide the target upstream chart version")
	},
	Run: getRebaseInfoHandler,
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

	printSubchartChecklist(rebaseInfoState)
}

// printSubchartChecklist prints a checklist of values.yaml entries to verify when rebasing,
// derived from the upstream chart's dependency appVersions.
func printSubchartChecklist(info rebase.ChartRebaseInfo) {
	if len(info.DependencyChartVersions) == 0 {
		return
	}

	fmt.Println("")
	fmt.Println(
		text.Color.Sprintf(text.FgYellow, "Subchart values.yaml tag checklist (verify these in your Rancher chart patches):"),
	)

	for _, dep := range info.DependencyChartVersions {
		normalized := monsubcharts.NormalizeName(dep.Name)
		if !monsubcharts.SubchartsToCheck[normalized] {
			continue
		}
		if dep.AppVersion == "" {
			continue
		}

		fmt.Printf("  %s (appVersion: %s):\n", dep.Name, dep.AppVersion)
		for _, rule := range monsubcharts.GetRules(normalized) {
			fmt.Printf("    → %s: %s\n", rule.ValuesKey, rule.Apply(dep.AppVersion))
		}
	}
	fmt.Println("")
}

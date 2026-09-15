package charts

import (
	"fmt"
	"os"

	"github.com/jedib0t/go-pretty/table"
	"github.com/jedib0t/go-pretty/text"
	"github.com/rancher/ob-charts-tool/cmd/groups"
	"github.com/rancher/ob-charts-tool/internal/cmd/preparerelease"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var prepareReleaseCmd = &cobra.Command{
	Use:     "prepareRelease",
	GroupID: groups.ChartsGroup.ID,
	Short:   "Prepare the Rancher Charts automation core releases for ORBS charts",
	Long: `Prepare the Rancher Charts automation core releases for ORBS charts.

Prerequisites:
  - The automation directory must already be on a branch ready for changes.
  - All updates for the specified Rancher versions will be committed to that branch.
  - The charts directory will have branches checked out automatically (dev-v2.X).`,
	Args: func(_ *cobra.Command, _ []string) error {
		return nil
	},
	Run: getPrepareReleaseHandler,
}

func init() {
	prepareReleaseCmd.Flags().String("chart-dir", "", "Pick where the Charts repo is cloned")
	prepareReleaseCmd.Flags().String("chart-automation-dir", "", "Pick where the Chart's repo automation branch is cloned (must already be on the target branch)")
	prepareReleaseCmd.Flags().StringSlice("rancher-minor", []string{}, "Rancher minor version(s) to process (e.g., '2.15' or '2.15,2.16' or multiple --rancher-minor flags)")
	prepareReleaseCmd.Flags().String("chart-remote", "origin", "Remote name for the charts repository")

	prepareReleaseCmd.MarkFlagRequired("chart-dir")
	prepareReleaseCmd.MarkFlagRequired("chart-automation-dir")
	prepareReleaseCmd.MarkFlagRequired("rancher-minor")
}

func getPrepareReleaseHandler(cmd *cobra.Command, _ []string) {
	chartDir, err := cmd.Flags().GetString("chart-dir")
	if err != nil {
		log.Fatalf("Failed to get chart-dir flag: %v", err)
	}

	automationDir, err := cmd.Flags().GetString("chart-automation-dir")
	if err != nil {
		log.Fatalf("Failed to get chart-automation-dir flag: %v", err)
	}

	rancherMinors, err := cmd.Flags().GetStringSlice("rancher-minor")
	if err != nil {
		log.Fatalf("Failed to get rancher-minor flag: %v", err)
	}

	if len(rancherMinors) == 0 {
		log.Fatal("At least one Rancher minor version must be specified")
	}

	chartRemote, err := cmd.Flags().GetString("chart-remote")
	if err != nil {
		log.Fatalf("Failed to get chart-remote flag: %v", err)
	}

	fmt.Println(
		text.AlignCenter.Apply(
			text.Color.Sprintf(text.FgBlue, "Preparing ORBS Releases"),
			80,
		),
	)
	fmt.Println()

	// Verify automation directory is on a branch and ready for changes
	currentBranch, err := preparerelease.GetCurrentBranch(automationDir)
	if err != nil {
		log.Fatalf("Failed to get current branch in automation directory: %v", err)
	}
	log.Infof("Automation directory is on branch: %s", currentBranch)
	fmt.Println(text.Color.Sprintf(text.FgCyan, "All changes will be saved to automation branch: %s", currentBranch))
	fmt.Println()

	log.WithFields(log.Fields{
		"chart-dir":         chartDir,
		"automation-dir":    automationDir,
		"automation-branch": currentBranch,
		"rancher-minors":    rancherMinors,
		"chart-remote":      chartRemote,
	}).Debug("Configuration")

	// Process each Rancher minor version
	for i, rancherMinor := range rancherMinors {
		if i > 0 {
			fmt.Println()
			fmt.Println(text.Color.Sprint(text.FgMagenta, "═════════════════════════════════════════════════════════════════════════════════"))
			fmt.Println()
		}

		fmt.Println(
			text.AlignCenter.Apply(
				text.Color.Sprintf(text.FgCyan, "Processing Rancher %s (%d/%d)", rancherMinor, i+1, len(rancherMinors)),
				80,
			),
		)
		fmt.Println()

		if err := processRancherMinor(chartDir, automationDir, rancherMinor, chartRemote); err != nil {
			log.Errorf("Failed to process Rancher %s: %v", rancherMinor, err)
			continue
		}
	}

	fmt.Println()
	fmt.Println(
		text.AlignCenter.Apply(
			text.Color.Sprintf(text.FgGreen, "✓ Successfully processed %d Rancher version(s)", len(rancherMinors)),
			80,
		),
	)
}

func processRancherMinor(chartDir, automationDir, rancherMinor, chartRemote string) error {
	chartFilter := preparerelease.ORBSChartFilter()

	// Step 0: Checkout the correct charts branch
	fmt.Println(text.Color.Sprint(text.FgYellow, "→ Step 0: Checking out charts branch..."))

	chartsBranch := preparerelease.RancherMinorToChartsBranch(rancherMinor)
	log.Infof("Charts branch for Rancher %s: %s", rancherMinor, chartsBranch)

	if err := preparerelease.EnsureGitBranch(chartDir, chartRemote, chartsBranch); err != nil {
		return fmt.Errorf("failed to checkout charts branch: %w", err)
	}

	fmt.Println()

	// Step 1: Filter ORBS charts from the charts directory
	fmt.Println(text.Color.Sprint(text.FgYellow, "→ Step 1: Discovering ORBS charts..."))
	orbsReleases := preparerelease.FilterORBSCharts(chartFilter, chartDir)
	log.Infof("Found %d ORBS charts", len(orbsReleases))

	// Step 2: Filter by Rancher minor version
	fmt.Println(text.Color.Sprintf(text.FgYellow, "→ Step 2: Filtering for Rancher %s...", rancherMinor))
	orbsReleases = preparerelease.FilterChartsByRancherMinor(rancherMinor, orbsReleases)
	log.Infof("After filtering: %d charts match Rancher %s", len(orbsReleases), rancherMinor)

	if len(orbsReleases) == 0 {
		log.Warnf("No charts found for Rancher %s", rancherMinor)
		return nil
	}

	// Step 3: Select highest versions
	fmt.Println(text.Color.Sprint(text.FgYellow, "→ Step 3: Selecting highest versions..."))
	newestCharts, err := preparerelease.SelectHighestVersions(orbsReleases)
	if err != nil {
		return fmt.Errorf("failed to select highest versions: %w", err)
	}

	// Display selected versions in a table
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Chart", "Version"})
	for chart, version := range newestCharts {
		t.AppendRow(table.Row{chart, version})
	}
	t.SetStyle(table.StyleLight)
	t.Render()
	fmt.Println()

	// Step 4: Find and load the automation release chart for this Rancher version
	fmt.Println(text.Color.Sprint(text.FgYellow, "→ Step 4: Loading automation release chart..."))
	automationFilePath, err := preparerelease.FindReleaseFileForMinor(automationDir, rancherMinor)
	if err != nil {
		return fmt.Errorf("failed to find release file for Rancher %s: %w", rancherMinor, err)
	}
	log.Infof("Found: %s", automationFilePath)

	automationChart, err := preparerelease.LoadAutomationReleaseChart(automationFilePath)
	if err != nil {
		return fmt.Errorf("failed to load automation chart: %w", err)
	}
	log.Debugf("Loaded automation chart with %d total charts", len(automationChart))

	// Step 5: Update ORBS charts with new versions (preserving all other charts)
	fmt.Println(text.Color.Sprint(text.FgYellow, "→ Step 5: Updating ORBS charts with new versions..."))
	automationChart = preparerelease.UpdateAutomationReleaseChart(automationChart, newestCharts, chartFilter)
	log.Info("ORBS charts updated, all other charts preserved")

	// Step 6: Save the updated automation chart
	fmt.Println(text.Color.Sprint(text.FgYellow, "→ Step 6: Saving changes..."))
	err = preparerelease.SaveAutomationReleaseChart(automationFilePath, automationChart)
	if err != nil {
		return fmt.Errorf("failed to save automation chart: %w", err)
	}

	fmt.Println()
	fmt.Println(
		text.AlignCenter.Apply(
			text.Color.Sprintf(text.FgGreen, "✓ Successfully updated automation release chart for Rancher %s", rancherMinor),
			80,
		),
	)
	fmt.Println(text.Color.Sprintf(text.FgCyan, "Updated file: %s", automationFilePath))

	return nil
}

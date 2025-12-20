package monitoring

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rancher/ob-charts-tool/cmd/groups"
	monitoringTest "github.com/rancher/ob-charts-tool/internal/cmd/testNewMonitoringVersion"
)

var (
	rancherURL   string
	sessionToken string
)

// testNewVersionCmd represents the testNewVersion command
var testNewVersionCmd = &cobra.Command{
	Use:     "testNewVersion",
	Short:   "Tests a new monitoring chart version against the previous one to identify dashboard regressions and fixes.",
	GroupID: groups.MonitoringGroup.ID,
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) == 1 {
			return nil
		}
		return fmt.Errorf("you must provide the target upstream chart version")
	},
	RunE: testNewMonitoringVersion,
}

func init() {
	testNewVersionCmd.Flags().StringVar(&rancherURL, "rancher-url", "https://localhost:8443", "Rancher URL")
	testNewVersionCmd.Flags().StringVar(&sessionToken, "rancher-token", "", "Rancher session token")
	testNewVersionCmd.MarkFlagRequired("rancher-token")
}

func testNewMonitoringVersion(cmd *cobra.Command, args []string) error {
	newVersion := args[0]

	// 1. Get previous version
	fmt.Printf("Looking for version previous to %s...\n", newVersion)
	previousVersion, err := monitoringTest.GetPreviousVersion(newVersion, rancherURL, sessionToken)
	if err != nil {
		return fmt.Errorf("error getting previous version: %w", err)
	}
	if previousVersion == "" {
		return fmt.Errorf("no previous version found for %s. Cannot perform comparison", newVersion)
	}
	fmt.Printf("Found previous version: %s\n", previousVersion)

	// 2. Test previous version
	fmt.Printf("\n--- Testing Previous Version: %s ---\n", previousVersion)
	previousVersionResults, err := testVersion(previousVersion, rancherURL, sessionToken)
	if err != nil {
		return fmt.Errorf("failed to test version %s: %w", previousVersion, err)
	}

	// 3. Test new version
	fmt.Printf("\n--- Testing New Version: %s ---\n", newVersion)
	newVersionResults, err := testVersion(newVersion, rancherURL, sessionToken)
	if err != nil {
		return fmt.Errorf("failed to test version %s: %w", newVersion, err)
	}

	// 4. Compare results
	fmt.Printf("\n--- Comparing Results ---\n")
	return compareResults(previousVersionResults, newVersionResults)
}

// testVersion is a helper to install, test, and uninstall a specific chart version.
func testVersion(version, rancherURL, sessionToken string) (map[string][]monitoringTest.PanelTestResult, error) {
	// Install
	fmt.Printf("Installing rancher-monitoring version %s...\n", version)
	if err := monitoringTest.InstallCurrentVersion(version, rancherURL, sessionToken); err != nil {
		return nil, fmt.Errorf("installation failed: %w", err)
	}
	fmt.Println("Installation complete. Waiting 1 minute for components to stabilize...")
	time.Sleep(1 * time.Minute)

	// Get and Test Dashboards
	fmt.Println("Getting dashboards...")
	dashboards, err := monitoringTest.GetDashboards()
	if err != nil {
		return nil, fmt.Errorf("could not get dashboards: %w", err)
	}

	allResults := make(map[string][]monitoringTest.PanelTestResult)
	for name, dashboard := range dashboards {
		fmt.Printf("Testing dashboard: %s\n", name)
		results, err := monitoringTest.TestDashboard(dashboard.(map[string]interface{}), rancherURL, sessionToken)
		if err != nil {
			fmt.Printf("  WARNING: Failed to test dashboard %s: %v\n", name, err)
			continue
		}
		allResults[name] = results
	}
	return allResults, nil
}

// uninstallChart is a helper to uninstall both monitoring charts.
func uninstallChart(version, rancherURL, sessionToken string) error {
	fmt.Printf("Uninstalling rancher-monitoring for version %s...\n", version)
	if err := monitoringTest.UninstallChart("rancher-monitoring", "cattle-monitoring-system", rancherURL, sessionToken); err != nil {
		return fmt.Errorf("failed to uninstall rancher-monitoring: %w", err)
	}
	fmt.Println("Uninstalling rancher-monitoring-crd...")
	if err := monitoringTest.UninstallChart("rancher-monitoring-crd", "cattle-monitoring-system", rancherURL, sessionToken); err != nil {
		return fmt.Errorf("failed to uninstall rancher-monitoring-crd: %w", err)
	}
	fmt.Println("Uninstallation complete.")
	return nil
}

// compareResults analyzes the test outcomes and prints any regressions or fixes.
func compareResults(prevResults, newResults map[string][]monitoringTest.PanelTestResult) error {
	regressionsFound := 0
	fixesFound := 0

	// Standardize newResults keys to lowercase for case-insensitive lookup
	lowerNewResults := make(map[string][]monitoringTest.PanelTestResult)
	for key, val := range newResults {
		lowerNewResults[strings.ToLower(key)] = val
	}

	for dashName, prevDashResults := range prevResults {
		newDashResults, ok := lowerNewResults[strings.ToLower(dashName)]
		if !ok {
			fmt.Printf("[WARNING] Dashboard '%s' is missing in the new version.\n", dashName)
			continue
		}

		prevPanelMap := make(map[string]monitoringTest.PanelTestResult)
		for _, p := range prevDashResults {
			prevPanelMap[strings.ToLower(p.Panel)] = p
		}
		newPanelMap := make(map[string]monitoringTest.PanelTestResult)
		for _, p := range newDashResults {
			newPanelMap[strings.ToLower(p.Panel)] = p
		}

		for panelNameKey, prevPanel := range prevPanelMap {
			newPanel, ok := newPanelMap[panelNameKey]
			if !ok {
				fmt.Printf("[WARNING] Dashboard '%s': Panel '%s' is missing in the new version.\n", dashName, prevPanel.Panel)
				continue
			}

			prevQueryMap := make(map[string]monitoringTest.QueryResult)
			for _, q := range prevPanel.Results {
				prevQueryMap[q.Expr] = q
			}

			for _, newQuery := range newPanel.Results {
				prevQuery, ok := prevQueryMap[newQuery.Expr]
				if !ok {
					// This is a new query, not a regression or fix.
					continue
				}

				// Check for Regressions
				if prevQuery.Status == "success" && newQuery.Status != "success" {
					fmt.Printf("[REGRESSION] Dashboard '%s', Panel '%s': Query failed in new version (was success).\n  - Query: %s\n  - New Error: %s\n", dashName, prevPanel.Panel, newQuery.Expr, newQuery.Error)
					regressionsFound++
				}
				if prevQuery.DataInResult && !newQuery.DataInResult {
					fmt.Printf("[REGRESSION] Dashboard '%s', Panel '%s': Query returned no data in new version (had data before).\n  - Query: %s\n", dashName, prevPanel.Panel, newQuery.Expr)
					regressionsFound++
				}

				// Check for Fixes
				if prevQuery.Status != "success" && newQuery.Status == "success" {
					fmt.Printf("[FIX] Dashboard '%s', Panel '%s': Query now succeeds (was failing).\n  - Query: %s\n", dashName, prevPanel.Panel, newQuery.Expr)
					fixesFound++
				}
				if !prevQuery.DataInResult && newQuery.DataInResult {
					fmt.Printf("[FIX] Dashboard '%s', Panel '%s': Query now returns data (previously empty).\n  - Query: %s\n", dashName, prevPanel.Panel, newQuery.Expr)
					fixesFound++
				}
			}
		}
	}

	fmt.Println("\n--- Summary ---")
	if regressionsFound == 0 && fixesFound == 0 {
		fmt.Println("No changes detected between versions.")
		return nil
	}

	fmt.Printf("Found %d regressions and %d fixes.\n", regressionsFound, fixesFound)
	if regressionsFound > 0 {
		return fmt.Errorf("test failed with %d regressions", regressionsFound)
	}
	return nil
}

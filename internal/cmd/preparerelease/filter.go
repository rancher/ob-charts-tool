package preparerelease

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/rancher/ob-charts-tool/helmtools/util"
)

type ChartFilterItem struct {
	Name        string
	HasCRDChart bool
}

var orbsCharts = []ChartFilterItem{
	{
		"rancher-backup",
		true,
	},
	{Name: "prometheus-federator"},
	{Name: "rancher-alerting-drivers"},
	{
		"rancher-logging",
		true,
	},
	{
		"rancher-monitoring",
		true,
	},
	{Name: "rancher-monitoring-dashboards"},
}

func ORBSCharts() []string {
	var allChartNames []string

	for _, filter := range orbsCharts {
		allChartNames = append(allChartNames, filter.Name)
		if filter.HasCRDChart {
			allChartNames = append(allChartNames, filter.Name+"-crd")
		}
	}

	return allChartNames
}

func ORBSChartFilter() func(string) bool {
	allChartNames := ORBSCharts()

	return func(chart string) bool {
		return slices.Contains(allChartNames, chart)
	}
}

// Something that maps against Chart prefix versions and rancher minors
// The constant is Rancher 2.10.z => 105.y.z with every other version being logically consistent
// As output we would expect just the major version part so 2.10 (anything) gets 105; 2.11 anything gets 106, etc
func rancherChartsMajorVersion(rancherMinor string) string {
	// Parse the minor version from the input string (e.g., "2.10" -> 10)
	parts := strings.Split(rancherMinor, ".")
	if len(parts) < 2 {
		return ""
	}

	// Extract the minor version number
	var minor int
	_, err := fmt.Sscanf(parts[1], "%d", &minor)
	if err != nil {
		return ""
	}

	// Calculate chart major version: 95 + minor
	// 2.10 -> 105, 2.11 -> 106, etc.
	chartMajor := 95 + minor

	return strconv.Itoa(chartMajor)
}

func ChartsMatchingRancherMinorFilter(minor string) (func(string) bool, error) {
	chartsVersionPrefix := rancherChartsMajorVersion(minor)
	if chartsVersionPrefix == "" {
		return nil, fmt.Errorf("invalid rancher minor version: %s", minor)
	}
	// Require complete major component match (e.g., "105." not just "105")
	// This prevents "105" from matching "1050.x.x"
	return func(chartVersion string) bool {
		return strings.HasPrefix(chartVersion, chartsVersionPrefix+".")
	}, nil
}

func FilterChartsByRancherMinor(minor string, releases map[string][]string) (map[string][]string, error) {
	chartMinorFilter, err := ChartsMatchingRancherMinorFilter(minor)
	if err != nil {
		return nil, err
	}
	for chart, versions := range releases {
		filtered := util.FilterSlice(versions, chartMinorFilter)
		if len(filtered) == 0 {
			delete(releases, chart)
		} else {
			releases[chart] = filtered
		}
	}

	return releases, nil
}

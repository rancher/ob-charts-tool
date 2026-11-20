package preparerelease

import (
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

type AllReleaseLineCharts map[string]ReleaseLineCharts
type ReleaseLineCharts map[string][]string

type ReleaseInfo struct {
	ToRelease bool `yaml:"ToRelease"`
	QA        bool `yaml:"QA"`
	UnRC      bool `yaml:"UnRC"`
	Released  bool `yaml:"Released"`
}

type AutomationReleaseChart map[string]map[string]ReleaseInfo

func FindAllORBSReleaseLines(filter func(string) bool, dir string) (AllReleaseLineCharts, error) {
	releaseVersionFiles, err := findRancherReleaseFiles(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to find release version files: %w", err)
	}

	res := make(AllReleaseLineCharts)
	for _, file := range releaseVersionFiles {
		// Extract version from file name (e.g., "2.9.0.yaml" -> "2.9.0")
		// Filenames use full 3-slot semver without 'v' prefix
		version := filepath.Base(file)
		version = version[:len(version)-len(filepath.Ext(version))]

		// Call FindORBSReleaseLineCharts to filter this release file
		releaseLineCharts, err := FindORBSReleaseLineCharts(filter, file)
		if err != nil {
			return nil, fmt.Errorf("failed to process release file %s: %w", file, err)
		}

		// Only save if there are ORBS charts in this release
		if len(releaseLineCharts) > 0 {
			res[version] = releaseLineCharts
		}
	}

	return res, nil
}

func findRancherReleaseFiles(dir string) ([]string, error) {
	searchPath := filepath.Join(dir, "release", "*.yaml")
	return filepath.Glob(searchPath)
}

// FindReleaseFileForMinor finds the release file that matches the given Rancher minor version.
// For example, rancherMinor "2.15" would match "2.15.0.yaml" or "2.15.1.yaml".
func FindReleaseFileForMinor(dir string, rancherMinor string) (string, error) {
	files, err := findRancherReleaseFiles(dir)
	if err != nil {
		return "", fmt.Errorf("failed to find release files: %w", err)
	}

	prefix := rancherMinor + "."
	for _, file := range files {
		baseName := filepath.Base(file)
		baseName = baseName[:len(baseName)-len(filepath.Ext(baseName))]

		if len(baseName) >= len(prefix) && baseName[:len(prefix)] == prefix {
			return file, nil
		}
	}

	return "", fmt.Errorf("no release file found for Rancher minor version %s", rancherMinor)
}

func FindORBSReleaseLineCharts(filter func(string) bool, filePath string) (ReleaseLineCharts, error) {
	releaseYaml, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer releaseYaml.Close()

	var release ReleaseLineCharts
	decoder := yaml.NewDecoder(releaseYaml)
	if err := decoder.Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode yaml: %w", err)
	}

	// Filter for ORBS charts
	filtered := make(ReleaseLineCharts)
	for chart, versionsInfo := range release {
		if filter(chart) {
			filtered[chart] = versionsInfo
		}
	}

	return filtered, nil
}

// UpdateAutomationReleaseChart updates an automation release chart with new versions for ORBS charts only.
// For each ORBS chart in versionMap, it replaces the <version> placeholder with the actual version
// and sets ToRelease to true. All other charts in the automation chart are preserved unchanged.
func UpdateAutomationReleaseChart(chart AutomationReleaseChart, versionMap map[string]string, filter func(string) bool) AutomationReleaseChart {
	result := make(AutomationReleaseChart)

	for chartName, versions := range chart {
		newVersions := make(map[string]ReleaseInfo)

		// Check if this is an ORBS chart and we have a new version for it
		if filter(chartName) {
			if newVersion, hasUpdate := versionMap[chartName]; hasUpdate {
				// Check if this version already exists in the automation chart
				if existingInfo, exists := versions[newVersion]; exists {
					// Version already exists - preserve existing flags but ensure ToRelease is true
					existingInfo.ToRelease = true
					newVersions[newVersion] = existingInfo
				} else {
					// New version - initialize with ToRelease: true
					newVersions[newVersion] = ReleaseInfo{
						ToRelease: true,
						QA:        false,
						UnRC:      false,
						Released:  false,
					}
				}
			} else {
				// ORBS chart but no update, copy existing versions
				for version, info := range versions {
					newVersions[version] = info
				}
			}
		} else {
			// Not an ORBS chart, preserve existing versions unchanged
			for version, info := range versions {
				newVersions[version] = info
			}
		}

		result[chartName] = newVersions
	}

	return result
}

// LoadAutomationReleaseChart loads an automation release chart from a YAML file.
func LoadAutomationReleaseChart(filePath string) (AutomationReleaseChart, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	var chart AutomationReleaseChart
	if err := yaml.Unmarshal(data, &chart); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML from %s: %w", filePath, err)
	}

	return chart, nil
}

// SaveAutomationReleaseChart writes an automation release chart to a YAML file.
func SaveAutomationReleaseChart(filePath string, chart AutomationReleaseChart) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2) // Use 2-space indentation to match original format
	defer encoder.Close()

	if err := encoder.Encode(chart); err != nil {
		return fmt.Errorf("failed to encode automation release chart: %w", err)
	}

	return nil
}

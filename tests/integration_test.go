package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rancher/ob-charts-tool/internal/cmd/preparerelease"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

// TestCase defines the structure for integration test cases.
// For successful test cases, both InputFile and ExpectedFile are required.
// For error test cases, only InputFile is required and ExpectError should be true.
type TestCase struct {
	Name         string
	InputFile    string
	ExpectedFile string
	ExpectError  bool
	ErrorMessage string
}

// loadReleaseYAML loads a YAML fixture file into a map[string][]string.
// This is used for fixtures that represent chart names mapped to version arrays.
func loadReleaseYAML(t *testing.T, filepath string) map[string][]string {
	t.Helper()

	data, err := os.ReadFile(filepath)
	require.NoError(t, err, "failed to read file: %s", filepath)

	var result map[string][]string
	err = yaml.Unmarshal(data, &result)
	require.NoError(t, err, "failed to unmarshal YAML from: %s", filepath)

	return result
}

// loadVersionMapYAML loads a YAML fixture file into a map[string]string.
// This is used for expected output fixtures that represent chart names mapped to single versions.
func loadVersionMapYAML(t *testing.T, filepath string) map[string]string {
	t.Helper()

	data, err := os.ReadFile(filepath)
	require.NoError(t, err, "failed to read file: %s", filepath)

	var result map[string]string
	err = yaml.Unmarshal(data, &result)
	require.NoError(t, err, "failed to unmarshal YAML from: %s", filepath)

	return result
}

// compareReleaseConfigs performs a deep comparison of two release configs.
// It checks that both maps have the same keys and that each chart's version array matches.
func compareReleaseConfigs(t *testing.T, got, want map[string][]string) {
	t.Helper()

	assert.Equal(t, len(want), len(got), "number of charts should match")

	for chartName, wantVersions := range want {
		gotVersions, exists := got[chartName]
		assert.True(t, exists, "chart %s should exist in result", chartName)
		if exists {
			assert.Equal(t, wantVersions, gotVersions, "versions for chart %s should match", chartName)
		}
	}

	for chartName := range got {
		_, exists := want[chartName]
		assert.True(t, exists, "unexpected chart %s in result", chartName)
	}
}

// TestFilterReleaseIntegration tests the FilterORBSCharts function using fixture files.
// It validates that only ORBS charts are retained from a full release.yaml.
func TestFilterReleaseIntegration(t *testing.T) {
	testCases := []TestCase{
		{
			Name:         "filter ORBS charts from full release",
			InputFile:    "testdata/test_filter_release_01_input.yaml",
			ExpectedFile: "testdata/test_filter_release_01_expected.yaml",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			input := loadReleaseYAML(t, tc.InputFile)
			expected := loadReleaseYAML(t, tc.ExpectedFile)

			chartFilter := preparerelease.ORBSChartFilter()

			// Create a temporary directory to write the test input
			tmpDir := t.TempDir()
			inputYAMLPath := filepath.Join(tmpDir, "release.yaml")

			// Write the input data to the temp directory
			inputData, err := yaml.Marshal(input)
			require.NoError(t, err, "failed to marshal input YAML")
			err = os.WriteFile(inputYAMLPath, inputData, 0644)
			require.NoError(t, err, "failed to write input YAML")

			// Call FilterORBSCharts with the temp directory
			result := preparerelease.FilterORBSCharts(chartFilter, tmpDir)

			// Compare results
			compareReleaseConfigs(t, result, expected)

			// Verify we have exactly 9 ORBS charts
			assert.Equal(t, 9, len(result), "should have exactly 9 ORBS charts")
		})
	}
}

// TestFilterVersionsIntegration tests the SelectHighestVersions function using fixture files.
// It validates that the highest version is selected from each chart's version list
// and that RC/beta/alpha suffixes are properly stripped.
func TestFilterVersionsIntegration(t *testing.T) {
	testCases := []TestCase{
		{
			Name:         "select highest versions and strip RC suffixes",
			InputFile:    "testdata/test_filter_versions_03_input.yaml",
			ExpectedFile: "testdata/test_filter_versions_03_expected.yaml",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			input := loadReleaseYAML(t, tc.InputFile)
			expected := loadVersionMapYAML(t, tc.ExpectedFile)

			result, err := preparerelease.SelectHighestVersions(input)
			require.NoError(t, err, "SelectHighestVersions should not error")

			// Compare each chart's selected version
			assert.Equal(t, len(expected), len(result), "number of charts should match")

			for chartName, wantVersion := range expected {
				gotVersion, exists := result[chartName]
				assert.True(t, exists, "chart %s should exist in result", chartName)
				if exists {
					assert.Equal(t, wantVersion, gotVersion, "version for chart %s should match", chartName)
				}
			}

			for chartName := range result {
				_, exists := expected[chartName]
				assert.True(t, exists, "unexpected chart %s in result", chartName)
			}
		})
	}
}

// TestFindORBSReleaseLineChartsIntegration tests the FindORBSReleaseLineCharts function using fixture files.
// It validates that the function correctly reads and filters a release file for ORBS charts.
func TestFindORBSReleaseLineChartsIntegration(t *testing.T) {
	testCases := []TestCase{
		{
			Name:         "find ORBS charts from release file",
			InputFile:    "testdata/test_filter_release_01_input.yaml",
			ExpectedFile: "testdata/test_filter_release_01_expected.yaml",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			expected := loadReleaseYAML(t, tc.ExpectedFile)

			chartFilter := preparerelease.ORBSChartFilter()

			// Call FindORBSReleaseLineCharts directly with the input file
			result, err := preparerelease.FindORBSReleaseLineCharts(chartFilter, tc.InputFile)
			require.NoError(t, err, "FindORBSReleaseLineCharts should not error")

			// Compare results
			compareReleaseConfigs(t, result, expected)

			// Verify we have exactly 9 ORBS charts
			assert.Equal(t, 9, len(result), "should have exactly 9 ORBS charts")
		})
	}
}

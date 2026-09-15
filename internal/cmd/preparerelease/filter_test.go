package preparerelease

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestORBSCharts(t *testing.T) {
	charts := ORBSCharts()

	// Should have 9 total charts (6 base + 3 CRD charts)
	assert.Equal(t, 9, len(charts), "should have 9 ORBS charts total")

	// Check that all base charts are present
	assert.Contains(t, charts, "rancher-backup")
	assert.Contains(t, charts, "prometheus-federator")
	assert.Contains(t, charts, "rancher-alerting-drivers")
	assert.Contains(t, charts, "rancher-logging")
	assert.Contains(t, charts, "rancher-monitoring")
	assert.Contains(t, charts, "rancher-monitoring-dashboards")

	// Check that CRD charts are present
	assert.Contains(t, charts, "rancher-backup-crd")
	assert.Contains(t, charts, "rancher-logging-crd")
	assert.Contains(t, charts, "rancher-monitoring-crd")
}

func TestORBSChartFilter(t *testing.T) {
	filter := ORBSChartFilter()

	// Should match ORBS charts
	assert.True(t, filter("rancher-backup"))
	assert.True(t, filter("rancher-backup-crd"))
	assert.True(t, filter("rancher-monitoring"))
	assert.True(t, filter("prometheus-federator"))

	// Should not match non-ORBS charts
	assert.False(t, filter("longhorn"))
	assert.False(t, filter("neuvector"))
	assert.False(t, filter("fleet"))
	assert.False(t, filter("random-chart"))
}

func TestRancherChartsMajorVersion(t *testing.T) {
	tests := []struct {
		name          string
		rancherMinor  string
		expectedMajor string
	}{
		{
			name:          "Rancher 2.10",
			rancherMinor:  "2.10",
			expectedMajor: "105",
		},
		{
			name:          "Rancher 2.11",
			rancherMinor:  "2.11",
			expectedMajor: "106",
		},
		{
			name:          "Rancher 2.15",
			rancherMinor:  "2.15",
			expectedMajor: "110",
		},
		{
			name:          "Rancher 2.16",
			rancherMinor:  "2.16",
			expectedMajor: "111",
		},
		{
			name:          "Invalid - only one part",
			rancherMinor:  "2",
			expectedMajor: "",
		},
		{
			name:          "Invalid - non-numeric minor",
			rancherMinor:  "2.abc",
			expectedMajor: "",
		},
		{
			name:          "Invalid - empty string",
			rancherMinor:  "",
			expectedMajor: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rancherChartsMajorVersion(tt.rancherMinor)
			assert.Equal(t, tt.expectedMajor, result)
		})
	}
}

func TestChartsMatchingRancherMinorFilter(t *testing.T) {
	tests := []struct {
		name         string
		rancherMinor string
		version      string
		shouldMatch  bool
	}{
		{
			name:         "2.15 matches 110.x.x",
			rancherMinor: "2.15",
			version:      "110.0.1+up7.0.1-rc.2",
			shouldMatch:  true,
		},
		{
			name:         "2.15 matches 110.0.0",
			rancherMinor: "2.15",
			version:      "110.0.0+up11.0.0",
			shouldMatch:  true,
		},
		{
			name:         "2.15 does not match 109.x.x",
			rancherMinor: "2.15",
			version:      "109.0.0",
			shouldMatch:  false,
		},
		{
			name:         "2.11 matches 106.x.x",
			rancherMinor: "2.11",
			version:      "106.0.1+up1.2.3",
			shouldMatch:  true,
		},
		{
			name:         "2.11 does not match 105.x.x",
			rancherMinor: "2.11",
			version:      "105.0.1+up1.2.3",
			shouldMatch:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := ChartsMatchingRancherMinorFilter(tt.rancherMinor)
			result := filter(tt.version)
			assert.Equal(t, tt.shouldMatch, result)
		})
	}
}

func TestFilterChartsByRancherMinor(t *testing.T) {
	tests := []struct {
		name         string
		rancherMinor string
		input        map[string][]string
		expected     map[string][]string
	}{
		{
			name:         "Filter for 2.15",
			rancherMinor: "2.15",
			input: map[string][]string{
				"rancher-monitoring": {
					"110.0.2+up80.9.1-rancher.20",
					"110.0.1+up80.9.1-rancher.19",
					"109.0.0+up80.9.1-rancher.18",
				},
				"prometheus-federator": {
					"110.0.1+up7.0.1-rc.2",
					"109.0.1+up7.0.1-rc.1",
				},
				"rancher-alerting-drivers": {
					"109.0.0",
				},
			},
			expected: map[string][]string{
				"rancher-monitoring": {
					"110.0.2+up80.9.1-rancher.20",
					"110.0.1+up80.9.1-rancher.19",
				},
				"prometheus-federator": {
					"110.0.1+up7.0.1-rc.2",
				},
				// rancher-alerting-drivers should be removed entirely
			},
		},
		{
			name:         "Empty input",
			rancherMinor: "2.15",
			input:        map[string][]string{},
			expected:     map[string][]string{},
		},
		{
			name:         "No matching versions",
			rancherMinor: "2.15",
			input: map[string][]string{
				"rancher-monitoring": {
					"109.0.0+up80.9.1-rancher.18",
				},
			},
			expected: map[string][]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterChartsByRancherMinor(tt.rancherMinor, tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

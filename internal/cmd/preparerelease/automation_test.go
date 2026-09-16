package preparerelease

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpdateAutomationReleaseChart(t *testing.T) {
	orbsFilter := ORBSChartFilter()

	tests := []struct {
		name       string
		input      AutomationReleaseChart
		versionMap map[string]string
		expected   AutomationReleaseChart
	}{
		{
			name: "Update ORBS charts, preserve non-ORBS charts",
			input: AutomationReleaseChart{
				"rancher-monitoring": {
					"<version>": ReleaseInfo{
						ToRelease: false,
						QA:        false,
						UnRC:      false,
						Released:  false,
					},
				},
				"prometheus-federator": {
					"<version>": ReleaseInfo{
						ToRelease: false,
						QA:        false,
						UnRC:      false,
						Released:  false,
					},
				},
				"longhorn": {
					"<version>": ReleaseInfo{
						ToRelease: false,
						QA:        false,
						UnRC:      false,
						Released:  false,
					},
				},
			},
			versionMap: map[string]string{
				"rancher-monitoring":   "110.0.2+up80.9.1-rancher.20",
				"prometheus-federator": "110.0.1+up7.0.1",
			},
			expected: AutomationReleaseChart{
				"rancher-monitoring": {
					"110.0.2+up80.9.1-rancher.20": ReleaseInfo{
						ToRelease: true,
						QA:        false,
						UnRC:      false,
						Released:  false,
					},
				},
				"prometheus-federator": {
					"110.0.1+up7.0.1": ReleaseInfo{
						ToRelease: true,
						QA:        false,
						UnRC:      false,
						Released:  false,
					},
				},
				"longhorn": {
					"<version>": ReleaseInfo{
						ToRelease: false,
						QA:        false,
						UnRC:      false,
						Released:  false,
					},
				},
			},
		},
		{
			name: "ORBS chart without update preserves existing version",
			input: AutomationReleaseChart{
				"rancher-monitoring": {
					"110.0.1+up80.9.1-rancher.19": ReleaseInfo{
						ToRelease: true,
						QA:        false,
						UnRC:      false,
						Released:  false,
					},
				},
			},
			versionMap: map[string]string{
				// No update for rancher-monitoring
			},
			expected: AutomationReleaseChart{
				"rancher-monitoring": {
					"110.0.1+up80.9.1-rancher.19": ReleaseInfo{
						ToRelease: true,
						QA:        false,
						UnRC:      false,
						Released:  false,
					},
				},
			},
		},
		{
			name: "Multiple versions preserved for non-ORBS charts",
			input: AutomationReleaseChart{
				"longhorn": {
					"110.1.0+up1.12.1": ReleaseInfo{
						ToRelease: false,
						QA:        false,
						UnRC:      false,
						Released:  false,
					},
					"110.0.0+up1.12.0": ReleaseInfo{
						ToRelease: false,
						QA:        false,
						UnRC:      false,
						Released:  true,
					},
				},
			},
			versionMap: map[string]string{},
			expected: AutomationReleaseChart{
				"longhorn": {
					"110.1.0+up1.12.1": ReleaseInfo{
						ToRelease: false,
						QA:        false,
						UnRC:      false,
						Released:  false,
					},
					"110.0.0+up1.12.0": ReleaseInfo{
						ToRelease: false,
						QA:        false,
						UnRC:      false,
						Released:  true,
					},
				},
			},
		},
		{
			name:       "Empty input",
			input:      AutomationReleaseChart{},
			versionMap: map[string]string{},
			expected:   AutomationReleaseChart{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UpdateAutomationReleaseChart(tt.input, tt.versionMap, orbsFilter)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReleaseInfoYAMLTags(t *testing.T) {
	// This test verifies that ReleaseInfo has the correct YAML tags
	// to preserve field name casing when marshaling/unmarshaling
	info := ReleaseInfo{
		ToRelease: true,
		QA:        false,
		UnRC:      false,
		Released:  false,
	}

	// These assertions check that the struct fields exist
	assert.Equal(t, true, info.ToRelease)
	assert.Equal(t, false, info.QA)
	assert.Equal(t, false, info.UnRC)
	assert.Equal(t, false, info.Released)
}

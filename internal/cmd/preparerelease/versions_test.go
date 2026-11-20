package preparerelease

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStripPrereleaseSuffix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "RC suffix after build metadata",
			input:    "110.0.2+up11.0.3-rc.4",
			expected: "110.0.2+up11.0.3",
		},
		{
			name:     "RC suffix before build metadata",
			input:    "110.0.2-rc.1+up80.9.1-rancher.20",
			expected: "110.0.2+up80.9.1-rancher.20",
		},
		{
			name:     "No RC suffix",
			input:    "110.0.1+up7.0.1",
			expected: "110.0.1+up7.0.1",
		},
		{
			name:     "Beta suffix",
			input:    "110.0.1-beta.1+up7.0.1",
			expected: "110.0.1+up7.0.1",
		},
		{
			name:     "Alpha suffix",
			input:    "110.0.1-alpha.2+up7.0.1",
			expected: "110.0.1+up7.0.1",
		},
		{
			name:     "Custom prerelease suffix (glorp)",
			input:    "110.0.2-glorp.1+up80.9.1-rancher.20",
			expected: "110.0.2+up80.9.1-rancher.20",
		},
		{
			name:     "Custom prerelease suffix (glop)",
			input:    "110.0.2+up11.0.3-glop.4",
			expected: "110.0.2+up11.0.3",
		},
		{
			name:     "Preserve rancher suffix in build metadata",
			input:    "110.0.0+up4.10.0-rancher.24",
			expected: "110.0.0+up4.10.0-rancher.24",
		},
		{
			name:     "Simple version without metadata",
			input:    "110.0.0",
			expected: "110.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripPrereleaseSuffix(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindHighestVersion(t *testing.T) {
	tests := []struct {
		name        string
		versions    []string
		expected    string
		expectError bool
	}{
		{
			name: "Multiple RC versions",
			versions: []string{
				"110.0.1+up7.0.1-rc.2",
				"110.0.1+up7.0.1-rc.1",
				"110.0.0+up7.0.0",
			},
			expected: "110.0.1+up7.0.1",
		},
		{
			name: "Mixed stable and prerelease",
			versions: []string{
				"110.0.2+up11.0.3-rc.4",
				"110.0.1+up11.0.2",
				"110.0.0+up11.0.0",
			},
			expected: "110.0.2+up11.0.3",
		},
		{
			name: "Single version",
			versions: []string{
				"110.0.1+up7.0.1",
			},
			expected: "110.0.1+up7.0.1",
		},
		{
			name: "Deduplication after prerelease stripping",
			versions: []string{
				"110.0.0-glorp.1+up4.10.0-rancher.24",
				"110.0.0-glop.2+up4.10.0-rancher.24",
			},
			expected: "110.0.0+up4.10.0-rancher.24",
		},
		{
			name: "Higher version wins with different prerelease",
			versions: []string{
				"110.0.2-glop.4+up4.10.0-rancher.24",
				"110.0.0-glorp.1+up4.10.0-rancher.25",
				"110.0.0-glop.2+up4.10.0-rancher.24",
			},
			expected: "110.0.2+up4.10.0-rancher.24",
		},
		{
			name: "First occurrence when build metadata differs",
			versions: []string{
				"110.0.2-glop.4+up4.10.0-rancher.27",
				"110.0.2-glop.4+up4.10.0-rancher.24",
				"110.0.0-glorp.1+up4.10.0-rancher.25",
				"110.0.0-glop.2+up4.10.0-rancher.24",
			},
			expected: "110.0.2+up4.10.0-rancher.27",
		},
		{
			name:        "Empty list",
			versions:    []string{},
			expectError: true,
		},
		{
			name: "Invalid version format",
			versions: []string{
				"not-a-version",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := findHighestVersion(tt.versions)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSelectHighestVersions(t *testing.T) {
	tests := []struct {
		name        string
		input       map[string][]string
		expected    map[string]string
		expectError bool
	}{
		{
			name: "Multiple charts with versions",
			input: map[string][]string{
				"prometheus-federator": {
					"110.0.1+up7.0.1-rc.2",
					"110.0.1+up7.0.1-rc.1",
				},
				"rancher-backup": {
					"110.0.2+up11.0.3-rc.4",
					"110.0.1+up11.0.2",
					"110.0.0+up11.0.0",
				},
				"rancher-monitoring": {
					"110.0.2-rc.1+up80.9.1-rancher.20",
					"110.0.1+up80.9.1-rancher.19",
					"110.0.0+up80.9.1-rancher.18",
				},
			},
			expected: map[string]string{
				"prometheus-federator": "110.0.1+up7.0.1",
				"rancher-backup":       "110.0.2+up11.0.3",
				"rancher-monitoring":   "110.0.2+up80.9.1-rancher.20",
			},
		},
		{
			name:     "Empty chart map",
			input:    map[string][]string{},
			expected: map[string]string{},
		},
		{
			name: "Chart with empty version list",
			input: map[string][]string{
				"rancher-backup": {},
			},
			expected: map[string]string{},
		},
		{
			name: "Chart with invalid version",
			input: map[string][]string{
				"rancher-backup": {
					"not-a-valid-version",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SelectHighestVersions(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

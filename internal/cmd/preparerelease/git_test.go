package preparerelease

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRancherMinorToChartsBranch(t *testing.T) {
	tests := []struct {
		name         string
		rancherMinor string
		expected     string
	}{
		{
			name:         "Rancher 2.15",
			rancherMinor: "2.15",
			expected:     "dev-v2.15",
		},
		{
			name:         "Rancher 2.16",
			rancherMinor: "2.16",
			expected:     "dev-v2.16",
		},
		{
			name:         "Rancher 2.10",
			rancherMinor: "2.10",
			expected:     "dev-v2.10",
		},
		{
			name:         "Rancher 3.0",
			rancherMinor: "3.0",
			expected:     "dev-v3.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RancherMinorToChartsBranch(tt.rancherMinor)
			assert.Equal(t, tt.expected, result)
		})
	}
}

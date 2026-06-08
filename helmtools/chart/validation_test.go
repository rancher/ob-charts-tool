package chart_test

import (
	"context"
	"testing"

	"github.com/rancher/ob-charts-tool/helmtools/chart"
	"github.com/stretchr/testify/assert"
)

func TestParseChartYAML_Validation(t *testing.T) {
	t.Run("rejects empty data", func(t *testing.T) {
		_, err := chart.ParseChartYAML(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("rejects empty byte slice", func(t *testing.T) {
		_, err := chart.ParseChartYAML([]byte{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})
}

func TestClient_FetchChartYAML_Validation(t *testing.T) {
	t.Run("rejects empty URL", func(t *testing.T) {
		client := chart.NewClient(nil)
		_, err := client.FetchChartYAML(context.Background(), "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})
}

func TestFindDependencies_Validation(t *testing.T) {
	t.Run("handles nil chart gracefully", func(t *testing.T) {
		deps := chart.FindDependencies(nil)
		assert.NotNil(t, deps)
		assert.Empty(t, deps)
	})

	t.Run("returns empty slice for chart with no dependencies", func(t *testing.T) {
		c := &chart.Chart{
			Dependencies: nil,
		}
		deps := chart.FindDependencies(c)
		assert.NotNil(t, deps)
		assert.Empty(t, deps)
	})
}

package values_test

import (
	"testing"

	"github.com/rancher/ob-charts-tool/helmtools/values"
	"github.com/stretchr/testify/assert"
)

func TestGetByPath_Validation(t *testing.T) {
	t.Run("returns false for nil data", func(t *testing.T) {
		value, found := values.GetByPath(nil, "image.tag")
		assert.False(t, found)
		assert.Empty(t, value)
	})

	t.Run("returns false for empty key path", func(t *testing.T) {
		data := map[string]interface{}{"key": "value"}
		value, found := values.GetByPath(data, "")
		assert.False(t, found)
		assert.Empty(t, value)
	})
}

func TestGetMapByPath_Validation(t *testing.T) {
	t.Run("returns false for nil data", func(t *testing.T) {
		result, found := values.GetMapByPath(nil, "image")
		assert.False(t, found)
		assert.Nil(t, result)
	})

	t.Run("returns false for empty key path", func(t *testing.T) {
		data := map[string]interface{}{"key": map[string]interface{}{"nested": "value"}}
		result, found := values.GetMapByPath(data, "")
		assert.False(t, found)
		assert.Nil(t, result)
	})
}

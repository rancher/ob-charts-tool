package util

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterSlice(t *testing.T) {
	t.Run("filter integers - keep even numbers", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5, 6}
		result := FilterSlice(input, func(n int) bool { return n%2 == 0 })
		assert.Equal(t, []int{2, 4, 6}, result)
	})

	t.Run("filter integers - keep numbers > 5", func(t *testing.T) {
		input := []int{1, 3, 5, 7, 9, 11}
		result := FilterSlice(input, func(n int) bool { return n > 5 })
		assert.Equal(t, []int{7, 9, 11}, result)
	})

	t.Run("filter strings - keep non-empty", func(t *testing.T) {
		input := []string{"hello", "", "world", "", "test"}
		result := FilterSlice(input, func(s string) bool { return s != "" })
		assert.Equal(t, []string{"hello", "world", "test"}, result)
	})

	t.Run("filter strings - keep strings with prefix", func(t *testing.T) {
		input := []string{"test-1", "foo", "test-2", "bar", "test-3"}
		result := FilterSlice(input, func(s string) bool { return strings.HasPrefix(s, "test-") })
		assert.Equal(t, []string{"test-1", "test-2", "test-3"}, result)
	})

	t.Run("empty slice returns empty", func(t *testing.T) {
		input := []int{}
		result := FilterSlice(input, func(n int) bool { return n > 0 })
		assert.Equal(t, []int{}, result)
		assert.NotNil(t, result, "should return empty slice, not nil")
	})

	t.Run("no matches returns empty slice", func(t *testing.T) {
		input := []int{1, 2, 3}
		result := FilterSlice(input, func(n int) bool { return n > 10 })
		assert.Equal(t, []int{}, result)
		assert.NotNil(t, result, "should return empty slice, not nil")
	})

	t.Run("all match returns all elements", func(t *testing.T) {
		input := []int{2, 4, 6, 8}
		result := FilterSlice(input, func(n int) bool { return n%2 == 0 })
		assert.Equal(t, []int{2, 4, 6, 8}, result)
	})

	t.Run("filter custom structs", func(t *testing.T) {
		type Person struct {
			Name string
			Age  int
		}
		input := []Person{
			{"Alice", 25},
			{"Bob", 17},
			{"Charlie", 30},
			{"Dave", 16},
		}
		result := FilterSlice(input, func(p Person) bool { return p.Age >= 18 })
		expected := []Person{
			{"Alice", 25},
			{"Charlie", 30},
		}
		assert.Equal(t, expected, result)
	})
}

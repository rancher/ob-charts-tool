package util

import (
	"encoding/json"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestNewSet(t *testing.T) {
	t.Run("create string set", func(t *testing.T) {
		s := NewSet[string]()
		assert.NotNil(t, s)
		assert.True(t, s.IsEmpty())
		assert.Equal(t, 0, s.Size())
	})

	t.Run("create int set", func(t *testing.T) {
		s := NewSet[int]()
		assert.NotNil(t, s)
		assert.True(t, s.IsEmpty())
	})
}

func TestSet_Add(t *testing.T) {
	t.Run("add string to set", func(t *testing.T) {
		s := NewSet[string]()
		err := s.Add("hello")
		assert.NoError(t, err)
		assert.True(t, s.Contains("hello"))
		assert.Equal(t, 1, s.Size())
	})

	t.Run("add multiple unique items", func(t *testing.T) {
		s := NewSet[int]()
		require.NoError(t, s.Add(1))
		require.NoError(t, s.Add(2))
		require.NoError(t, s.Add(3))
		assert.Equal(t, 3, s.Size())
		assert.True(t, s.Contains(1))
		assert.True(t, s.Contains(2))
		assert.True(t, s.Contains(3))
	})

	t.Run("add duplicate items - set behavior", func(t *testing.T) {
		s := NewSet[string]()
		require.NoError(t, s.Add("duplicate"))
		require.NoError(t, s.Add("duplicate"))
		require.NoError(t, s.Add("duplicate"))
		assert.Equal(t, 1, s.Size(), "set should only contain one instance")
		assert.True(t, s.Contains("duplicate"))
	})

	t.Run("cannot add empty string", func(t *testing.T) {
		s := NewSet[string]()
		err := s.Add("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot add empty value")
		assert.Equal(t, 0, s.Size())
	})

	t.Run("cannot add zero value int", func(t *testing.T) {
		s := NewSet[int]()
		err := s.Add(0)
		assert.Error(t, err)
		assert.Equal(t, 0, s.Size())
	})

	t.Run("can add non-zero values", func(t *testing.T) {
		s := NewSet[int]()
		require.NoError(t, s.Add(1))
		require.NoError(t, s.Add(-1))
		require.NoError(t, s.Add(100))
		assert.Equal(t, 3, s.Size())
	})
}

func TestSet_Contains(t *testing.T) {
	t.Run("contains existing item", func(t *testing.T) {
		s := NewSet[string]()
		require.NoError(t, s.Add("exists"))
		assert.True(t, s.Contains("exists"))
	})

	t.Run("does not contain missing item", func(t *testing.T) {
		s := NewSet[string]()
		require.NoError(t, s.Add("exists"))
		assert.False(t, s.Contains("missing"))
	})

	t.Run("empty set contains nothing", func(t *testing.T) {
		s := NewSet[int]()
		assert.False(t, s.Contains(1))
		assert.False(t, s.Contains(0))
	})
}

func TestSet_Remove(t *testing.T) {
	t.Run("remove existing item", func(t *testing.T) {
		s := NewSet[string]()
		require.NoError(t, s.Add("remove-me"))
		assert.True(t, s.Contains("remove-me"))
		s.Remove("remove-me")
		assert.False(t, s.Contains("remove-me"))
		assert.Equal(t, 0, s.Size())
	})

	t.Run("remove non-existent item is safe", func(t *testing.T) {
		s := NewSet[string]()
		require.NoError(t, s.Add("keep"))
		s.Remove("not-there")
		assert.Equal(t, 1, s.Size())
		assert.True(t, s.Contains("keep"))
	})

	t.Run("remove from empty set is safe", func(t *testing.T) {
		s := NewSet[int]()
		s.Remove(42)
		assert.Equal(t, 0, s.Size())
	})

	t.Run("remove multiple items", func(t *testing.T) {
		s := NewSet[int]()
		require.NoError(t, s.Add(1))
		require.NoError(t, s.Add(2))
		require.NoError(t, s.Add(3))
		s.Remove(1)
		s.Remove(3)
		assert.Equal(t, 1, s.Size())
		assert.True(t, s.Contains(2))
		assert.False(t, s.Contains(1))
		assert.False(t, s.Contains(3))
	})
}

func TestSet_Map(t *testing.T) {
	t.Run("map strings to uppercase", func(t *testing.T) {
		s := NewSet[string]()
		require.NoError(t, s.Add("hello"))
		require.NoError(t, s.Add("world"))

		result := s.Map(func(str string) string {
			return str + "!"
		})

		assert.Equal(t, 2, result.Size())
		assert.True(t, result.Contains("hello!"))
		assert.True(t, result.Contains("world!"))
		// Original set unchanged
		assert.True(t, s.Contains("hello"))
		assert.True(t, s.Contains("world"))
	})

	t.Run("map integers multiply by 2", func(t *testing.T) {
		s := NewSet[int]()
		require.NoError(t, s.Add(1))
		require.NoError(t, s.Add(2))
		require.NoError(t, s.Add(3))

		result := s.Map(func(n int) int { return n * 2 })

		assert.Equal(t, 3, result.Size())
		assert.True(t, result.Contains(2))
		assert.True(t, result.Contains(4))
		assert.True(t, result.Contains(6))
	})

	t.Run("map to same value creates single-element set", func(t *testing.T) {
		s := NewSet[int]()
		require.NoError(t, s.Add(1))
		require.NoError(t, s.Add(2))
		require.NoError(t, s.Add(3))

		result := s.Map(func(_ int) int { return 42 })

		assert.Equal(t, 1, result.Size(), "all mapped to same value")
		assert.True(t, result.Contains(42))
	})

	t.Run("map empty set returns empty set", func(t *testing.T) {
		s := NewSet[string]()
		result := s.Map(func(str string) string { return str + "!" })
		assert.Equal(t, 0, result.Size())
		assert.True(t, result.IsEmpty())
	})
}

func TestSet_Values(t *testing.T) {
	t.Run("values returns all items", func(t *testing.T) {
		s := NewSet[string]()
		require.NoError(t, s.Add("apple"))
		require.NoError(t, s.Add("banana"))
		require.NoError(t, s.Add("cherry"))

		values := s.Values()
		assert.Len(t, values, 3)
		sort.Strings(values)
		assert.Equal(t, []string{"apple", "banana", "cherry"}, values)
	})

	t.Run("empty set returns empty slice", func(t *testing.T) {
		s := NewSet[int]()
		values := s.Values()
		assert.Equal(t, []int{}, values)
		assert.NotNil(t, values)
	})

	t.Run("values are independent copy", func(t *testing.T) {
		s := NewSet[int]()
		require.NoError(t, s.Add(1))
		values := s.Values()
		values[0] = 999
		// Original set unchanged
		assert.True(t, s.Contains(1))
		assert.False(t, s.Contains(999))
	})
}

func TestSet_ValuesChan(t *testing.T) {
	t.Run("channel delivers all items", func(t *testing.T) {
		s := NewSet[int]()
		require.NoError(t, s.Add(1))
		require.NoError(t, s.Add(2))
		require.NoError(t, s.Add(3))

		received := make([]int, 0)
		for val := range s.ValuesChan() {
			received = append(received, val)
		}

		assert.Len(t, received, 3)
		sort.Ints(received)
		assert.Equal(t, []int{1, 2, 3}, received)
	})

	t.Run("empty set closes channel immediately", func(t *testing.T) {
		s := NewSet[string]()
		count := 0
		for range s.ValuesChan() {
			count++
		}
		assert.Equal(t, 0, count)
	})
}

func TestSet_Size(t *testing.T) {
	t.Run("empty set has size 0", func(t *testing.T) {
		s := NewSet[string]()
		assert.Equal(t, 0, s.Size())
	})

	t.Run("size increases with adds", func(t *testing.T) {
		s := NewSet[int]()
		assert.Equal(t, 0, s.Size())
		require.NoError(t, s.Add(1))
		assert.Equal(t, 1, s.Size())
		require.NoError(t, s.Add(2))
		assert.Equal(t, 2, s.Size())
	})

	t.Run("size decreases with removes", func(t *testing.T) {
		s := NewSet[int]()
		require.NoError(t, s.Add(1))
		require.NoError(t, s.Add(2))
		assert.Equal(t, 2, s.Size())
		s.Remove(1)
		assert.Equal(t, 1, s.Size())
		s.Remove(2)
		assert.Equal(t, 0, s.Size())
	})

	t.Run("duplicate adds don't increase size", func(t *testing.T) {
		s := NewSet[string]()
		require.NoError(t, s.Add("same"))
		require.NoError(t, s.Add("same"))
		require.NoError(t, s.Add("same"))
		assert.Equal(t, 1, s.Size())
	})
}

func TestSet_IsEmpty(t *testing.T) {
	t.Run("new set is empty", func(t *testing.T) {
		s := NewSet[int]()
		assert.True(t, s.IsEmpty())
	})

	t.Run("set with items is not empty", func(t *testing.T) {
		s := NewSet[string]()
		require.NoError(t, s.Add("item"))
		assert.False(t, s.IsEmpty())
	})

	t.Run("set becomes empty after removing all items", func(t *testing.T) {
		s := NewSet[int]()
		require.NoError(t, s.Add(42))
		assert.False(t, s.IsEmpty())
		s.Remove(42)
		assert.True(t, s.IsEmpty())
	})
}

func TestIsEmpty(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		assert.True(t, IsEmpty(""))
	})

	t.Run("non-empty string", func(t *testing.T) {
		assert.False(t, IsEmpty("hello"))
	})

	t.Run("zero int", func(t *testing.T) {
		assert.True(t, IsEmpty(0))
	})

	t.Run("non-zero int", func(t *testing.T) {
		assert.False(t, IsEmpty(42))
		assert.False(t, IsEmpty(-1))
	})

	t.Run("empty struct", func(t *testing.T) {
		type Empty struct{}
		assert.True(t, IsEmpty(Empty{}))
	})

	t.Run("non-empty struct", func(t *testing.T) {
		type Person struct {
			Name string
		}
		assert.True(t, IsEmpty(Person{}))
		assert.False(t, IsEmpty(Person{Name: "Alice"}))
	})

	t.Run("nil slice", func(t *testing.T) {
		var s []int
		assert.True(t, IsEmpty(s))
	})

	t.Run("empty slice", func(t *testing.T) {
		s := []int{}
		assert.False(t, IsEmpty(s), "empty slice is not zero value (nil)")
	})
}

func TestSet_MarshalJSON(t *testing.T) {
	t.Run("serialize set to JSON array", func(t *testing.T) {
		s := NewSet[string]()
		require.NoError(t, s.Add("apple"))
		require.NoError(t, s.Add("banana"))

		data, err := json.Marshal(s)
		require.NoError(t, err)

		// Unmarshal to verify it's an array
		var result []string
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		sort.Strings(result)
		assert.Equal(t, []string{"apple", "banana"}, result)
	})

	t.Run("empty set serializes to empty array", func(t *testing.T) {
		s := NewSet[int]()
		data, err := json.Marshal(s)
		require.NoError(t, err)
		assert.Equal(t, "[]", string(data))
	})
}

func TestSet_MarshalYAML(t *testing.T) {
	t.Run("serialize set to YAML array", func(t *testing.T) {
		s := NewSet[string]()
		require.NoError(t, s.Add("one"))
		require.NoError(t, s.Add("two"))

		data, err := yaml.Marshal(s)
		require.NoError(t, err)

		// Unmarshal to verify
		var result []string
		err = yaml.Unmarshal(data, &result)
		require.NoError(t, err)

		sort.Strings(result)
		assert.Equal(t, []string{"one", "two"}, result)
	})

	t.Run("empty set serializes to empty YAML array", func(t *testing.T) {
		s := NewSet[string]()
		data, err := yaml.Marshal(s)
		require.NoError(t, err)
		assert.Contains(t, string(data), "[]\n")
	})
}

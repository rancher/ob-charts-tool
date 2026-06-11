package values

import (
	"fmt"
	"strings"
)

// GetByPath follows a dotted key path (e.g. "image.tag") through a parsed YAML map
// and returns the string value at that path.
func GetByPath(data map[string]interface{}, keyPath string) (string, bool) {
	if data == nil || keyPath == "" {
		return "", false
	}
	parts := strings.Split(keyPath, ".")
	current := data
	for i, part := range parts {
		val, ok := current[part]
		if !ok {
			return "", false
		}
		if i == len(parts)-1 {
			// Last part - return as string
			if s, ok := val.(string); ok {
				return s, true
			}
			return fmt.Sprintf("%v", val), true
		}
		// Not last part - must be a map
		next, ok := val.(map[string]interface{})
		if !ok {
			return "", false
		}
		current = next
	}
	return "", false
}

// GetMapByPath follows a dotted key path through a parsed YAML map and returns the
// nested map at that path. Useful for navigating to an image struct (e.g. "kubeRBACProxy.image").
func GetMapByPath(data map[string]interface{}, keyPath string) (map[string]interface{}, bool) {
	if data == nil || keyPath == "" {
		return nil, false
	}
	parts := strings.Split(keyPath, ".")
	current := data
	for _, part := range parts {
		val, ok := current[part]
		if !ok {
			return nil, false
		}
		next, ok := val.(map[string]interface{})
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

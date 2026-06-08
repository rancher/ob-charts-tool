package values

import "strings"

// NormalizeName strips the "rancher-" prefix from a subchart directory name.
// This is specific to the ob-charts-tool workflow where Rancher prefixes chart names.
func NormalizeName(name string) string {
	return strings.TrimPrefix(name, "rancher-")
}

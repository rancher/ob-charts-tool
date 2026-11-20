package preparerelease

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/rancher/ob-charts-tool/helmtools/util"
)

// prereleaseSuffixPattern matches prerelease identifiers like -rc.X, -beta.X, -alpha.X
// but preserves the build metadata (everything after +)
var prereleaseSuffixPattern = regexp.MustCompile(`-(?:rancher\.[0-9A-Za-z-]+|[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)`)

// SelectHighestVersions takes a map of chart names to version arrays and returns
// a map of chart names to their highest version with prerelease suffixes stripped.
//
// For each chart:
// 1. Parse all versions as semantic versions
// 2. Sort to find the highest version
// 3. Strip prerelease suffixes (-rc.X, -beta.X, -alpha.X)
// 4. Preserve build metadata (+upX.Y.Z-rancher.N)
//
// Example transformations:
//
//	110.0.2+up11.0.3-rc.4 → 110.0.2+up11.0.3
//	110.0.2-rc.1+up80.9.1-rancher.20 → 110.0.2+up80.9.1-rancher.20
//	110.0.1+up7.0.1-rc.2 → 110.0.1+up7.0.1
//	110.0.2-glorp.1+up80.9.1-rancher.20 → 110.0.2+up80.9.1-rancher.20
//	110.0.2+up11.0.3-glop.4 → 110.0.2+up11.0.3
func SelectHighestVersions(charts map[string][]string) (map[string]string, error) {
	result := make(map[string]string, len(charts))

	for chartName, versionList := range charts {
		if len(versionList) == 0 {
			continue
		}

		highestVersion, err := findHighestVersion(versionList)
		if err != nil {
			return nil, fmt.Errorf("failed to find highest version for chart %s: %w", chartName, err)
		}

		cleanedVersion := stripPrereleaseSuffix(highestVersion)
		result[chartName] = cleanedVersion
	}

	return result, nil
}

// findHighestVersion parses a list of version strings and returns the highest one.
// To avoid semver treating build metadata (+upX.Y.Z) as identical, we strip
// prerelease suffixes first, deduplicate, then sort. This ensures deterministic
// selection when multiple versions differ only in build metadata.
func findHighestVersion(versionList []string) (string, error) {
	if len(versionList) == 0 {
		return "", errors.New("version list is empty")
	}

	// Strip prerelease suffixes from all versions first
	cleanedVersions := make([]string, 0, len(versionList))
	for _, versionStr := range versionList {
		cleaned := stripPrereleaseSuffix(versionStr)
		cleanedVersions = append(cleanedVersions, cleaned)
	}

	// Deduplicate versions
	uniqueVersions := util.Unique(cleanedVersions)

	// Build version collection from deduplicated cleaned versions
	versions := make([]*semver.Version, 0, len(uniqueVersions))
	for _, cleaned := range uniqueVersions {
		v, err := semver.NewVersion(cleaned)
		if err != nil {
			return "", fmt.Errorf("failed to parse version %s: %w", cleaned, err)
		}
		versions = append(versions, v)
	}

	// Sort in reverse order to get highest first
	sort.Sort(sort.Reverse(semver.Collection(versions)))

	return versions[0].Original(), nil
}

// stripPrereleaseSuffix removes prerelease identifiers (-rc.X, -beta.X, -alpha.X)
// from a version string while preserving build metadata.
//
// Examples:
//
//	110.0.2+up11.0.3-rc.4 → 110.0.2+up11.0.3
//	110.0.2-rc.1+up80.9.1-rancher.20 → 110.0.2+up80.9.1-rancher.20
//	110.0.1+up7.0.1 → 110.0.1+up7.0.1 (no change)
func stripPrereleaseSuffix(versionStr string) string {
	// The pattern matches -rc.X, -beta.X, -alpha.X anywhere in the string
	// This handles both:
	// - Versions where prerelease comes before build metadata: 110.0.2-rc.1+up80.9.1
	// - Versions where prerelease comes after build metadata: 110.0.2+up11.0.3-rc.4
	return prereleaseSuffixPattern.ReplaceAllStringFunc(versionStr, func(match string) string {
		if strings.HasPrefix(match, "-rancher.") {
			return match // leave it untouched
		}
		return "" // strip it
	})
}

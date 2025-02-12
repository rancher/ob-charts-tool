package git

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/go-git/go-git/v5/plumbing"
	log "github.com/sirupsen/logrus"
)

// extractVersion extracts the semantic version from a git tag reference.
func extractVersion(ref *plumbing.Reference, componentPrefix string) (*semver.Version, error) {
	fullTag := ref.Name().String() // e.g., "refs/tags/component-1.2.3"
	prefix := "refs/tags/" + componentPrefix + "-"
	if !strings.HasPrefix(fullTag, prefix) {
		return nil, fmt.Errorf("unexpected tag format: %s", fullTag)
	}
	versionStr := strings.TrimPrefix(fullTag, prefix)
	return semver.NewVersion(versionStr)
}

// FindHighestVersionTag selects the tag with the highest version number.
func FindHighestVersionTag(tags []*plumbing.Reference, componentPrefix string) *plumbing.Reference {
	var highestRef *plumbing.Reference
	var highestVersion *semver.Version

	for _, tag := range tags {
		version, err := extractVersion(tag, componentPrefix)
		if err != nil {
			log.Printf("Skipping invalid tag: %v", err)
			continue
		}

		if highestVersion == nil || version.GreaterThan(highestVersion) {
			highestVersion = version
			highestRef = tag
		}
	}

	return highestRef
}

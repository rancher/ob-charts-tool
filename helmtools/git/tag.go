package git

import (
	"context"
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/rancher/ob-charts-tool/helmtools/util"
)

// VerifyTagExists checks if a tag exists in a remote repository.
// Returns (exists bool, ref string, hash string, error).
// The context parameter is currently unused but reserved for future use when go-git supports it.
func VerifyTagExists(_ context.Context, repoURL string, tag string) (bool, string, string, error) {
	if repoURL == "" {
		return false, "", "", fmt.Errorf("repository URL cannot be empty")
	}
	if tag == "" {
		return false, "", "", fmt.Errorf("tag cannot be empty")
	}
	remote := git.NewRemote(nil, &config.RemoteConfig{URLs: []string{repoURL}})
	// TODO: Pass context to List when go-git v5 supports it
	refs, err := remote.List(&git.ListOptions{})
	if err != nil {
		return false, "", "", fmt.Errorf("error listing remote refs: %w", err)
	}

	expectedTagRef := "refs/tags/" + tag
	for _, ref := range refs {
		if ref.Name().String() == expectedTagRef {
			return true, expectedTagRef, ref.Hash().String(), nil
		}
	}

	return false, expectedTagRef, "", nil
}

// FindMatchingTags finds all tags in a remote repository that contain the given string.
// Returns (found bool, matching tags, error).
// The context parameter is currently unused but reserved for future use when go-git supports it.
func FindMatchingTags(_ context.Context, repoURL string, tagPartial string) (bool, []Tag, error) {
	if repoURL == "" {
		return false, nil, fmt.Errorf("repository URL cannot be empty")
	}
	if tagPartial == "" {
		return false, nil, fmt.Errorf("tag pattern cannot be empty")
	}
	remote := git.NewRemote(nil, &config.RemoteConfig{URLs: []string{repoURL}})
	// TODO: Pass context to List when go-git v5 supports it
	refs, err := remote.List(&git.ListOptions{})
	if err != nil {
		return false, nil, fmt.Errorf("error listing remote refs: %w", err)
	}

	matchingRefs := util.FilterSlice(refs, func(reference *plumbing.Reference) bool {
		return strings.Contains(reference.Name().Short(), tagPartial)
	})

	if len(matchingRefs) == 0 {
		return false, nil, nil
	}

	// Convert to Tag structs
	tags := make([]Tag, len(matchingRefs))
	for i, ref := range matchingRefs {
		tags[i] = Tag{
			Name:       ref.Name().Short(),
			Ref:        ref.Name().String(),
			CommitHash: ref.Hash().String(),
		}
	}

	return true, tags, nil
}

// FindHighestVersionTag selects the tag with the highest semantic version number
// from the provided tags, filtering by the given prefix.
// Returns nil if no valid version tags are found.
func FindHighestVersionTag(tags []Tag, componentPrefix string) *Tag {
	if len(tags) == 0 {
		return nil
	}
	if componentPrefix == "" {
		return nil
	}
	var highestTag *Tag
	var highestVersion *semver.Version

	prefix := componentPrefix + "-"
	for i := range tags {
		tag := &tags[i]

		// Check if tag name starts with expected prefix
		if !strings.HasPrefix(tag.Name, prefix) {
			continue
		}

		// Extract version string
		versionStr := strings.TrimPrefix(tag.Name, prefix)
		version, err := semver.NewVersion(versionStr)
		if err != nil {
			continue // Skip invalid version
		}

		if highestVersion == nil || version.GreaterThan(highestVersion) {
			highestVersion = version
			highestTag = tag
		}
	}

	return highestTag
}

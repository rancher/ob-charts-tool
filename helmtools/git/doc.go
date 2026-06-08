// Package git provides utilities for querying Git repositories for Helm chart tags and versions.
//
// # Basic Usage
//
// Verify a specific tag exists:
//
//	exists, ref, hash, err := git.VerifyTagExists(ctx, repoURL, "v1.0.0")
//
// Find all tags matching a pattern:
//
//	found, tags, err := git.FindMatchingTags(ctx, repoURL, "kube-prometheus-")
//
// Find the highest version tag:
//
//	highestTag := git.FindHighestVersionTag(tags, "kube-prometheus-stack")
//
// All Git operations use go-git for remote repository access and do not require
// a local clone.
package git

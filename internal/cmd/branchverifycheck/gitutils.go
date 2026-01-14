package branchverifycheck

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	gitpkg "github.com/rancher/ob-charts-tool/internal/git"
	log "github.com/rancher/ob-charts-tool/internal/logging"
)

const (
	// ToolRemoteName is the tool-owned remote name used to ensure consistent upstream state
	ToolRemoteName = "_ob-verify-upstream"
	// CanonicalUpstreamURL is the source of truth for the upstream repository
	CanonicalUpstreamURL = "https://github.com/rancher/ob-team-charts.git"
)

// UpstreamInfo holds information about the upstream remote being used
type UpstreamInfo struct {
	RemoteName    string
	IsToolCreated bool // true if we created a temporary remote
}

// findExistingCanonicalRemote checks if the repo already has a remote pointing to canonical upstream.
// Returns the remote name if found, empty string otherwise.
func findExistingCanonicalRemote(repo *git.Repository) string {
	remotes, err := gitpkg.GetRemoteURLs(repo)
	if err != nil {
		return ""
	}

	for remoteName, urls := range remotes {
		for _, url := range urls {
			if isCanonicalUpstreamURL(url) {
				log.Log.Debugf("Found existing canonical remote '%s' with URL '%s'", remoteName, url)
				return remoteName
			}
		}
	}

	return ""
}

// isCanonicalUpstreamURL checks if a URL points to the canonical rancher/ob-team-charts repo.
func isCanonicalUpstreamURL(url string) bool {
	normalized := strings.TrimSuffix(url, ".git")
	normalized = strings.ToLower(normalized)

	if normalized == "https://github.com/rancher/ob-team-charts" {
		return true
	}
	if normalized == "git@github.com:rancher/ob-team-charts" {
		return true
	}

	return false
}

// EnsureUpstreamRemote ensures we have access to upstream state.
// It first checks if an existing remote points to canonical upstream.
// If not, it creates a temporary tool-owned remote.
// Returns the reference to upstream/main after fetching, and info about which remote was used.
func EnsureUpstreamRemote(repo *git.Repository, repoPath string) (*plumbing.Reference, *UpstreamInfo, error) {
	info := &UpstreamInfo{}

	// First, check if an existing remote already points to canonical upstream
	existingRemote := findExistingCanonicalRemote(repo)
	if existingRemote != "" {
		info.RemoteName = existingRemote
		info.IsToolCreated = false
		log.Log.Debugf("Using existing remote '%s' for upstream", existingRemote)
	} else {
		// No existing canonical remote, create our tool remote
		info.RemoteName = ToolRemoteName
		info.IsToolCreated = true

		// Check if our tool remote already exists (from a previous run)
		_, err := repo.Remote(ToolRemoteName)
		if err != nil {
			log.Log.Debugf("Creating tool remote '%s' pointing to %s", ToolRemoteName, CanonicalUpstreamURL)
			cmd := exec.Command("git", "remote", "add", ToolRemoteName, CanonicalUpstreamURL)
			cmd.Dir = repoPath
			if output, err := cmd.CombinedOutput(); err != nil {
				return nil, nil, fmt.Errorf("failed to add remote: %v, output: %s", err, string(output))
			}
		}
	}

	// Fetch from the remote to get latest state
	log.Log.Debugf("Fetching from '%s' to get latest upstream state", info.RemoteName)
	cmd := exec.Command("git", "fetch", info.RemoteName, "--quiet")
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, info, fmt.Errorf("failed to fetch from upstream: %v, output: %s", err, string(output))
	}

	// Get the main branch reference from the remote
	for _, refName := range []string{
		fmt.Sprintf("refs/remotes/%s/main", info.RemoteName),
		fmt.Sprintf("refs/remotes/%s/master", info.RemoteName),
	} {
		ref, err := repo.Reference(plumbing.ReferenceName(refName), true)
		if err == nil {
			log.Log.Debugf("Using ref '%s' as upstream reference", refName)
			return ref, info, nil
		}
	}

	return nil, info, fmt.Errorf("could not find main/master branch on upstream remote")
}

// CleanupToolRemote removes the temporary tool-created remote if it was created.
func CleanupToolRemote(repoPath string, info *UpstreamInfo) {
	if info == nil || !info.IsToolCreated {
		return
	}

	log.Log.Debugf("Cleaning up tool remote '%s'", ToolRemoteName)
	cmd := exec.Command("git", "remote", "remove", ToolRemoteName)
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Log.Debugf("Failed to remove tool remote (may not exist): %v, output: %s", err, string(output))
	}
}

// GetGitRefs retrieves all the git references needed for verification.
// It ensures the upstream remote exists and is up-to-date, then returns
// HEAD, upstream, and merge-base references.
// Also returns UpstreamInfo so the caller can clean up any temporary remote.
func GetGitRefs(repo *git.Repository, repoPath string) (*GitRefs, *UpstreamInfo, error) {
	refs := &GitRefs{}

	// Get HEAD reference
	headRef, err := repo.Head()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get HEAD: %w", err)
	}
	refs.HeadRef = headRef

	// Get HEAD commit
	headCommit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get HEAD commit: %w", err)
	}
	refs.HeadCommit = headCommit

	// Ensure upstream remote and get reference
	upstreamRef, upstreamInfo, err := EnsureUpstreamRemote(repo, repoPath)
	if err != nil {
		return nil, upstreamInfo, fmt.Errorf("failed to ensure upstream remote: %w", err)
	}
	refs.UpstreamRef = upstreamRef

	// Re-open repo to see the new refs (go-git caches refs)
	repo, err = git.PlainOpen(repoPath)
	if err != nil {
		return nil, upstreamInfo, fmt.Errorf("failed to reopen repo: %w", err)
	}

	// Get upstream commit
	upstreamCommit, err := repo.CommitObject(upstreamRef.Hash())
	if err != nil {
		return nil, upstreamInfo, fmt.Errorf("failed to get upstream commit: %w", err)
	}
	refs.UpstreamCommit = upstreamCommit

	// Find merge-base between HEAD and upstream
	mergeBaseCommits, err := headCommit.MergeBase(upstreamCommit)
	if err != nil {
		return nil, upstreamInfo, fmt.Errorf("failed to find merge-base: %w", err)
	}

	if len(mergeBaseCommits) == 0 {
		return nil, upstreamInfo, fmt.Errorf("no common ancestor found between branch and upstream/main")
	}

	refs.MergeBaseCommit = mergeBaseCommits[0]

	return refs, upstreamInfo, nil
}

// CountCommitsBehind counts how many commits the merge-base is behind upstream.
func CountCommitsBehind(repo *git.Repository, upstreamRef *plumbing.Reference, mergeBaseHash plumbing.Hash) int {
	commitsBehind := 0
	commitIter, err := repo.Log(&git.LogOptions{From: upstreamRef.Hash()})
	if err != nil {
		return 0
	}
	defer commitIter.Close()

	for {
		c, err := commitIter.Next()
		if err != nil || c.Hash == mergeBaseHash {
			break
		}
		commitsBehind++
	}

	return commitsBehind
}

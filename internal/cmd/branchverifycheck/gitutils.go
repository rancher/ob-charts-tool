package branchverifycheck

import (
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	gitpkg "github.com/rancher/ob-charts-tool/internal/git"
	"github.com/rancher/ob-charts-tool/internal/git/remote"
	log "github.com/rancher/ob-charts-tool/internal/logging"
)

const (
	// CanonicalUpstreamURL is the source of truth for the upstream repository
	CanonicalUpstreamURL = "https://github.com/rancher/ob-team-charts.git"
)

// findExistingCanonicalRemote checks if the repo already has a remote pointing to canonical upstream.
// Returns the remote name if found, empty string otherwise.
func findExistingCanonicalRemote(repo *git.Repository) string {
	remotes, err := gitpkg.GetRemoteURLs(repo)
	if err != nil {
		return ""
	}

	for remoteName, urls := range remotes {
		for _, url := range urls {
			if gitpkg.IsGitHubRepoURL(url, "rancher", "ob-team-charts") {
				log.Log.Debugf("Found existing canonical remote '%s' with URL '%s'", remoteName, url)
				return remoteName
			}
		}
	}

	return ""
}

// EnsureUpstreamRemote ensures we have access to upstream state.
// It creates a tool-owned remote with HTTPS URL to avoid SSH authentication issues.
// Returns the reference to upstream/main after fetching.
func EnsureUpstreamRemote(repo *git.Repository) (*plumbing.Reference, error) {
	// Check if an existing remote points to canonical upstream (for logging purposes)
	existingRemote := findExistingCanonicalRemote(repo)
	if existingRemote != "" {
		log.Log.Debugf("Found existing canonical remote '%s', but using tool remote for reliable HTTPS fetch", existingRemote)
	}

	// Always use our tool remote with HTTPS URL to avoid SSH auth issues
	log.Log.Debugf("Setting up tool remote for upstream fetch")
	_, err := remote.FetchFromURL(repo, CanonicalUpstreamURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from upstream: %w", err)
	}

	// Get the main branch reference from the remote
	ref, err := remote.GetRemoteRef(repo, remote.ToolRemoteName, "main", "master")
	if err != nil {
		return nil, fmt.Errorf("could not find main/master branch on upstream remote: %w", err)
	}

	log.Log.Debugf("Using ref '%s' as upstream reference", ref.Name())
	return ref, nil
}

// CleanupToolRemote removes the temporary tool-created remote if it was created.
func CleanupToolRemote(repo *git.Repository) {
	log.Log.Debugf("Cleaning up tool remote '%s'", remote.ToolRemoteName)
	remote.CleanupToolRemote(repo)
}

// GetGitRefs retrieves all the git references needed for verification.
// It ensures the upstream remote exists and is up-to-date, then returns
// HEAD, upstream, and merge-base references.
func GetGitRefs(repo *git.Repository, repoPath string) (*GitRefs, error) {
	refs := &GitRefs{}

	// Get HEAD reference
	headRef, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}
	refs.HeadRef = headRef

	// Get HEAD commit
	headCommit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD commit: %w", err)
	}
	refs.HeadCommit = headCommit

	// Ensure upstream remote and get reference
	upstreamRef, err := EnsureUpstreamRemote(repo)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure upstream remote: %w", err)
	}
	refs.UpstreamRef = upstreamRef

	// Re-open repo to see the new refs (go-git caches refs)
	repo, err = git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to reopen repo: %w", err)
	}

	// Get upstream commit
	upstreamCommit, err := repo.CommitObject(upstreamRef.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get upstream commit: %w", err)
	}
	refs.UpstreamCommit = upstreamCommit

	// Find merge-base between HEAD and upstream
	mergeBaseCommits, err := headCommit.MergeBase(upstreamCommit)
	if err != nil {
		return nil, fmt.Errorf("failed to find merge-base: %w", err)
	}

	if len(mergeBaseCommits) == 0 {
		return nil, fmt.Errorf("no common ancestor found between branch and upstream/main")
	}

	refs.MergeBaseCommit = mergeBaseCommits[0]

	return refs, nil
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

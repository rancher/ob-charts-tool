package remote

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	gitpkg "github.com/rancher/ob-charts-tool/internal/git"
	"github.com/rancher/ob-charts-tool/internal/util"
	log "github.com/sirupsen/logrus"
)

func VerifyTagExists(repoURL string, tag string) (bool, string, string) {
	remote := git.NewRemote(nil, &config.RemoteConfig{URLs: []string{
		repoURL,
	}})
	refs, err := remote.List(&git.ListOptions{})
	if err != nil {
		log.Fatalf("Error listing remote refs: %v", err)
	}

	// Check if the reference exists
	expectedTagRef := "refs/tags/" + tag
	found := false
	var hash string
	for _, ref := range refs {
		if ref.Name().String() == expectedTagRef {
			found = true
			hash = ref.Hash().String()
			fmt.Printf("Found reference: %s (%s)\n", ref.Name(), ref.Hash())
			break
		}
	}

	return found, expectedTagRef, hash
}

func FindTagsMatching(repoURL string, tagPartial string) (bool, []*plumbing.Reference) {
	remote := git.NewRemote(nil, &config.RemoteConfig{URLs: []string{
		repoURL,
	}})
	refs, err := remote.List(&git.ListOptions{})
	if err != nil {
		log.Fatalf("Error listing remote refs: %v", err)
	}

	refs = util.FilterSlice(refs, func(reference *plumbing.Reference) bool {
		return strings.Contains(reference.Name().Short(), tagPartial)
	})

	if len(refs) > 0 {
		return true, refs
	}

	return false, refs
}

// ToolRemoteName is the standard name for tool-created temporary remotes
const ToolRemoteName = "_ob-tool-remote"

// EnsureRemoteExists ensures a remote with the given name exists pointing to the URL.
// If a remote with that name already exists, it verifies the URL matches.
// Always normalizes URLs to HTTPS for consistent behavior.
func EnsureRemoteExists(repo *git.Repository, name string, url string) (*git.Remote, error) {
	// Normalize URL to HTTPS for consistent auth-free fetching
	httpsURL := gitpkg.NormalizeGitHubURL(url)

	// Check if remote already exists
	remote, err := repo.Remote(name)
	if err == nil {
		// Remote exists, verify URL
		urls := remote.Config().URLs
		if len(urls) > 0 && gitpkg.NormalizeGitHubURL(urls[0]) == httpsURL {
			return remote, nil
		}
		// URL doesn't match, delete and recreate
		if err := repo.DeleteRemote(name); err != nil {
			return nil, fmt.Errorf("failed to delete existing remote '%s': %w", name, err)
		}
	}

	// Create new remote with HTTPS URL
	remote, err = repo.CreateRemote(&config.RemoteConfig{
		Name: name,
		URLs: []string{httpsURL},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create remote '%s': %w", name, err)
	}

	return remote, nil
}

// FetchFromRemote fetches from a remote, handling the "already up-to-date" case gracefully.
// Uses HTTPS URLs to avoid SSH authentication issues with go-git.
func FetchFromRemote(remote *git.Remote) error {
	err := remote.Fetch(&git.FetchOptions{})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("failed to fetch: %w", err)
	}
	return nil
}

// FetchFromURL creates a temporary remote and fetches from a URL.
// This is useful when you need to fetch from a URL that may be in SSH format
// but want to use HTTPS to avoid authentication issues.
// The remote is created with the ToolRemoteName and should be cleaned up after use.
func FetchFromURL(repo *git.Repository, url string) (*git.Remote, error) {
	remote, err := EnsureRemoteExists(repo, ToolRemoteName, url)
	if err != nil {
		return nil, err
	}

	if err := FetchFromRemote(remote); err != nil {
		return nil, err
	}

	return remote, nil
}

// CleanupToolRemote removes the tool-created remote if it exists.
func CleanupToolRemote(repo *git.Repository) {
	_ = repo.DeleteRemote(ToolRemoteName)
}

// GetRemoteRef gets a reference from a remote after ensuring it's fetched.
// Tries common branch names (main, master) in order.
func GetRemoteRef(repo *git.Repository, remoteName string, branchNames ...string) (*plumbing.Reference, error) {
	if len(branchNames) == 0 {
		branchNames = []string{"main", "master"}
	}

	for _, branch := range branchNames {
		refName := plumbing.ReferenceName(fmt.Sprintf("refs/remotes/%s/%s", remoteName, branch))
		ref, err := repo.Reference(refName, true)
		if err == nil {
			return ref, nil
		}
	}

	return nil, fmt.Errorf("could not find any of branches %v on remote '%s'", branchNames, remoteName)
}

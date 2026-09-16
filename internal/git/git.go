package git

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/rancher/ob-charts-tool/internal/git/auth"
	log "github.com/rancher/ob-charts-tool/internal/logging"
)

func VerifyDirIsGitRepo(repoDir string) bool {
	_, err := git.PlainOpen(repoDir)
	return err == nil
}

type RepoQAHintInfo struct {
	Path           string
	CurrentBranch  string
	RemoteRepoName string
	RemoteRepoURL  string
}

func FindLocalRepoBranchAndRemote(dir string) (*RepoQAHintInfo, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return nil, err
	}

	branchName, err := FindRepoBranchName(repo)
	if err != nil {
		return nil, err
	}

	var remoteName, remoteBranchURL string
	if IsCurrentBranchLocalOnly(repo) {
		log.Log.Debug("remote branch url is empty; will try to guess based on default remote")
		defaultRemoteInfo, err := FindRepoDefaultRemoteURL(repo)
		// TODO maybe add CLI flag for strict error mode and error instaed?
		if err != nil {
			log.Log.Warnf("current branch is local only, not finding remote branch")
			remoteBranchURL = "<UNKNOWN>"
			remoteName = "<UNKNOWN>"
		} else {
			remoteBranchURL = defaultRemoteInfo.URL
			// TODO: if the name is too generic (origin/etc) we should extract from URL
			// We need a name here that is "unique" in context of developer the change comes from
			remoteName = defaultRemoteInfo.Name
		}
	} else {
		log.Log.Panicf("TODO: implement this to find Remote branch")
	}

	if strings.Contains(remoteBranchURL, "git@") {
		parts := strings.Split(remoteBranchURL, ":")
		repoName := parts[1]
		repoName = repoName[:len(repoName)-4]
		// Transform the Git URL to an HTTP URL
		remoteBranchURL = "https://github.com/" + repoName
	}

	if remoteName == "origin" {
		githubRepoName := strings.SplitAfter(remoteBranchURL, "github.com/")
		repoParts := strings.Split(githubRepoName[1], "/")
		remoteName = repoParts[0]
	}

	return &RepoQAHintInfo{
		Path:           dir,
		CurrentBranch:  branchName,
		RemoteRepoName: remoteName,
		RemoteRepoURL:  remoteBranchURL,
	}, nil
}

func FindRepoBranchName(repo *git.Repository) (string, error) {
	headRef, err := repo.Head()
	if err != nil {
		return "", err
	}

	if !headRef.Name().IsBranch() {
		return "", errors.New("current HEAD is not on a branch")
	}

	fullNameRef := headRef.Name()

	return fullNameRef.Short(), nil
}

func IsCurrentBranchLocalOnly(repo *git.Repository) bool {
	headRef, err := repo.Head()
	if err != nil {
		return true
	}

	// If HEAD is not a branch (e.g., detached HEAD pointing directly to a commit or a tag),
	// then it's not a "branch" in the sense of being pushed or not.
	// For this function's purpose, we'll return true as it's not a branch to be tracked.
	if !headRef.Name().IsBranch() {
		return true
	}

	currentBranchName := headRef.Name()

	// Iterate over all remotes
	remotes, err := repo.Remotes()
	if err != nil {
		log.Log.Infof("Error getting remotes: %v", err)
		return false
	}

	for _, remote := range remotes {
		// Get the remote's references (including remote-tracking branches)
		remoteRefs, err := remote.List(&git.ListOptions{})
		if err != nil {
			log.Log.Warnf("Error getting refs for remote %s: %v", remote.Config().Name, err)
			continue // Try the next remote
		}

		// Check if the current local branch has a corresponding remote-tracking branch
		// The remote-tracking branch would typically be "refs/remotes/<remote_name>/<branch_name>"
		remoteTrackingBranchName := plumbing.ReferenceName("refs/remotes/" + remote.Config().Name + "/" + currentBranchName.Short())

		if slices.ContainsFunc(remoteRefs, func(ref *plumbing.Reference) bool {
			return ref.Name() == remoteTrackingBranchName
		}) {
			// Found a corresponding remote-tracking branch, so it's not local-only
			return false
		}
	}

	// If we've checked all remotes and found no corresponding remote-tracking branch,
	// then the current branch is local-only.
	return true
}

type RemoteRef struct {
	Name string
	URL  string
}

func FindRepoDefaultRemoteName(repo *git.Repository) (string, error) {
	headRef, err := repo.Head()
	if err != nil {
		return "", err
	}

	branchRef, err := repo.Branch(headRef.Name().Short())
	if err != nil {
		return "", err
	}

	remoteName := branchRef.Remote
	if remoteName == "" {
		return "", errors.New("could not find remote branch from current HEAD")
	}

	return remoteName, nil
}

// FindRepoDefaultRemoteURL will find the default (origin) remote repo's URL
func FindRepoDefaultRemoteURL(repo *git.Repository) (*RemoteRef, error) {
	defaultRemoteName, err := FindRepoDefaultRemoteName(repo)
	if err != nil {
		return nil, err
	}

	remote, err := repo.Remote(defaultRemoteName)
	if err != nil {
		return nil, fmt.Errorf("failed to find remote '%s': %w", defaultRemoteName, err)
	}

	return &RemoteRef{
		Name: defaultRemoteName,
		URL:  remote.Config().URLs[0],
	}, nil
}

// GetRemoteURLs gets all remote URLs for a repository
func GetRemoteURLs(repo *git.Repository) (map[string][]string, error) {
	remotes, err := repo.Remotes()
	if err != nil {
		return nil, fmt.Errorf("failed to get remotes: %w", err)
	}

	remoteURLs := make(map[string][]string)
	for _, remote := range remotes {
		remoteConfig := remote.Config()
		var urls []string
		for _, u := range remoteConfig.URLs {
			urls = append(urls, u)
		}
		remoteURLs[remoteConfig.Name] = urls
	}
	return remoteURLs, nil
}

// FetchBranch fetches a specific branch from a remote using proper SSH/HTTPS auth.
// It resolves authentication the same way native git does.
func FetchBranch(repo *git.Repository, remoteName, branchName string) error {
	// Get the remote
	remote, err := repo.Remote(remoteName)
	if err != nil {
		return fmt.Errorf("failed to get remote %s: %w", remoteName, err)
	}

	if len(remote.Config().URLs) == 0 {
		return fmt.Errorf("remote %s has no URLs configured", remoteName)
	}
	remoteURL := remote.Config().URLs[0]

	// Resolve auth using the auth manager
	authMgr := auth.New()
	authMethod, err := authMgr.Resolve(remoteURL)
	if err != nil {
		return fmt.Errorf("failed to resolve auth for %s: %w", remoteURL, err)
	}

	// Fetch the specific branch
	refSpec := config.RefSpec(fmt.Sprintf("+refs/heads/%s:refs/remotes/%s/%s", branchName, remoteName, branchName))
	err = repo.Fetch(&git.FetchOptions{
		RemoteName: remoteName,
		RefSpecs:   []config.RefSpec{refSpec},
		Auth:       authMethod,
	})

	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to fetch %s from %s: %w", branchName, remoteName, err)
	}

	return nil
}

// CheckoutBranch checks out a branch, creating or resetting it to match a remote branch.
// If the branch doesn't exist locally, it creates it. If it exists, it resets it to match the remote.
func CheckoutBranch(repo *git.Repository, remoteName, branchName string, force bool) error {
	// Get the remote branch reference
	remoteBranchRef := plumbing.NewRemoteReferenceName(remoteName, branchName)
	remoteRef, err := repo.Reference(remoteBranchRef, true)
	if err != nil {
		return fmt.Errorf("remote branch %s/%s not found: %w", remoteName, branchName, err)
	}

	// Get worktree
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Local branch reference
	localBranchRef := plumbing.NewBranchReferenceName(branchName)

	// Check if the local branch exists
	_, err = repo.Reference(localBranchRef, false)
	branchExists := err == nil

	if branchExists {
		// Branch exists, update it to point to the remote commit first
		err = repo.Storer.SetReference(plumbing.NewHashReference(localBranchRef, remoteRef.Hash()))
		if err != nil {
			return fmt.Errorf("failed to update branch reference: %w", err)
		}

		// Then checkout the updated branch
		err = worktree.Checkout(&git.CheckoutOptions{
			Branch: localBranchRef,
			Force:  force,
		})
		if err != nil {
			return fmt.Errorf("failed to checkout existing branch: %w", err)
		}
	} else {
		// Branch doesn't exist, create it
		err = worktree.Checkout(&git.CheckoutOptions{
			Branch: localBranchRef,
			Hash:   remoteRef.Hash(),
			Force:  force,
			Create: true,
		})
		if err != nil {
			return fmt.Errorf("failed to create and checkout branch: %w", err)
		}
	}

	return nil
}

package git

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
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

	var remoteName, remoteBranchUrl string
	if IsCurrentBranchLocalOnly(repo) {
		log.Log.Debug("remote branch url is empty; will try to guess based on default remote")
		defaultRemoteInfo, err := FindRepoDefaultRemoteUrl(repo)
		// TODO maybe add CLI flag for strict error mode and error instaed?
		if err != nil {
			log.Log.Warnf("current branch is local only, not finding remote branch")
			remoteBranchUrl = "<UNKNOWN>"
			remoteName = "<UNKNOWN>"
		} else {
			remoteBranchUrl = defaultRemoteInfo.URL
			// TODO: if the name is too generic (origin/etc) we should extract from URL
			// We need a name here that is "unique" in context of developer the change comes from
			remoteName = defaultRemoteInfo.Name
		}
	} else {
		remoteBranchRef, err := FindRepoRemoteBranchURL(repo, branchName)
		if err != nil {
			log.Log.Warn(err)
			return nil, err
		}

		remoteBranchUrl = remoteBranchRef.URL
		remoteName = remoteBranchRef.Name
	}

	if strings.Contains(remoteBranchUrl, "git@") {
		parts := strings.Split(remoteBranchUrl, ":")
		repoName := parts[1]
		repoName = repoName[:len(repoName)-4]
		// Transform the Git URL to an HTTP URL
		remoteBranchUrl = fmt.Sprintf("https://github.com/%s", repoName)
	}

	if remoteName == "origin" {
		githubRepoName := strings.SplitAfter(remoteBranchUrl, "github.com/")
		repoParts := strings.Split(githubRepoName[1], "/")
		remoteName = repoParts[0]
	}

	return &RepoQAHintInfo{
		Path:           dir,
		CurrentBranch:  branchName,
		RemoteRepoName: remoteName,
		RemoteRepoURL:  remoteBranchUrl,
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

// FindRepoRemoteBranchURL will return the current branch (if on one) remote repo URL
func FindRepoRemoteBranchURL(repo *git.Repository, branch string) (*RemoteRef, error) {
	config, err := repo.Config()
	fmt.Println(config, err)

	return nil, err
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
		return "", fmt.Errorf("could not find remote branch from current HEAD")
	}

	return remoteName, nil
}

// FindRepoDefaultRemoteUrl will find the default (origin) remote repo's URL
func FindRepoDefaultRemoteUrl(repo *git.Repository) (*RemoteRef, error) {
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

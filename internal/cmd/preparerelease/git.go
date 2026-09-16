package preparerelease

import (
	"fmt"

	"github.com/go-git/go-git/v5"
	gitpkg "github.com/rancher/ob-charts-tool/internal/git"
	log "github.com/sirupsen/logrus"
)

// RancherMinorToChartsBranch converts a Rancher minor version to the corresponding charts branch name.
// For example, "2.15" -> "dev-v2.15"
func RancherMinorToChartsBranch(rancherMinor string) string {
	return "dev-v" + rancherMinor
}

// EnsureGitBranch ensures the repository is on the specified branch from the remote.
// It fetches the latest state of the branch from the remote and checks it out.
// Uses the internal/git package which handles SSH/HTTPS auth like native git.
//
// This operation is destructive: it resets the local branch to match the remote
// and discards any uncommitted changes. It will fail if the worktree is dirty.
func EnsureGitBranch(repoDir, remoteName, branchName string) error {
	log.Infof("Opening repository at %s", repoDir)
	repo, err := git.PlainOpen(repoDir)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	// Check for uncommitted changes before proceeding with destructive operations
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := worktree.Status()
	if err != nil {
		return fmt.Errorf("failed to get worktree status: %w", err)
	}

	if !status.IsClean() {
		return fmt.Errorf("worktree has uncommitted changes in %s - commit or stash them before running prepare-release", repoDir)
	}

	log.Infof("Fetching latest %s from %s", branchName, remoteName)
	if err := gitpkg.FetchBranch(repo, remoteName, branchName); err != nil {
		return err
	}

	log.Infof("Checking out %s/%s", remoteName, branchName)
	if err := gitpkg.CheckoutBranch(repo, remoteName, branchName, true); err != nil {
		return err
	}

	log.Infof("Successfully checked out %s/%s", remoteName, branchName)
	return nil
}

// GetCurrentBranch returns the current git branch name for the given repository.
func GetCurrentBranch(repoDir string) (string, error) {
	repo, err := git.PlainOpen(repoDir)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	branchName, err := gitpkg.FindRepoBranchName(repo)
	if err != nil {
		return "", err
	}

	return branchName, nil
}

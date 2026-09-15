package preparerelease

import (
	"fmt"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

// RancherMinorToChartsBranch converts a Rancher minor version to the corresponding charts branch name.
// For example, "2.15" -> "dev-v2.15"
func RancherMinorToChartsBranch(rancherMinor string) string {
	return fmt.Sprintf("dev-v%s", rancherMinor)
}

// EnsureGitBranch ensures the repository is on the specified branch from the remote.
// It fetches from the remote and checks out the branch.
func EnsureGitBranch(repoDir, remote, branch string) error {
	log.Infof("Fetching latest from %s in %s", remote, repoDir)

	// Fetch from remote
	fetchCmd := exec.Command("git", "fetch", remote)
	fetchCmd.Dir = repoDir
	if output, err := fetchCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to fetch from %s: %w\n%s", remote, err, string(output))
	}

	// Check if branch exists on remote
	remoteBranch := fmt.Sprintf("%s/%s", remote, branch)
	lsRemoteCmd := exec.Command("git", "ls-remote", "--heads", remote, branch)
	lsRemoteCmd.Dir = repoDir
	output, err := lsRemoteCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to check if branch %s exists on %s: %w", branch, remote, err)
	}

	if len(strings.TrimSpace(string(output))) == 0 {
		return fmt.Errorf("branch %s does not exist on remote %s", branch, remote)
	}

	log.Infof("Checking out %s", remoteBranch)

	// Checkout the branch from remote
	checkoutCmd := exec.Command("git", "checkout", "-B", branch, remoteBranch)
	checkoutCmd.Dir = repoDir
	if output, err := checkoutCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to checkout %s: %w\n%s", remoteBranch, err, string(output))
	}

	log.Infof("Successfully checked out %s", remoteBranch)
	return nil
}

// GetCurrentBranch returns the current git branch name for the given repository.
func GetCurrentBranch(repoDir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

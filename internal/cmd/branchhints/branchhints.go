package branchhints

import (
	"errors"
	"fmt"

	"github.com/rancher/ob-charts-tool/internal/git"
)

const (
	ClusterRepoName     = "%s-%s" // Where 1st %s is "repo org/owner" (or ob-team-charts if rancher), and 2nd %s is "branch name"
	ClusterRepoTemplate = `apiVersion: catalog.cattle.io/v1
kind: ClusterRepo
metadata:
  name: %s
spec:
  gitBranch: %s
  gitRepo: %s
`
)

func PrepareBranchHints(repoDir string) (string, error) {
	if !git.VerifyDirIsGitRepo(repoDir) {
		return "", errors.New("dir path provided is not a git repository")
	}

	branchRef, err := git.FindLocalRepoBranchAndRemote(repoDir)
	if err != nil {
		return "", err
	}
	fmt.Printf("branch ref: %s\n", branchRef)

	clusterRepoName := fmt.Sprintf(ClusterRepoName, branchRef.RemoteRepoName, branchRef.CurrentBranch)
	return fmt.Sprintf(ClusterRepoTemplate, clusterRepoName, branchRef.CurrentBranch, branchRef.RemoteRepoURL), nil
}

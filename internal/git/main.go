package git

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/rancher/ob-charts-tool/internal/util"
	log "github.com/sirupsen/logrus"
	"strings"
)

func VerifyTagExists(repo string, tag string) (bool, string, string) {
	remote := git.NewRemote(nil, &config.RemoteConfig{URLs: []string{
		repo,
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

func FindTagsMatching(repoUrl string, tagPartial string) (bool, []*plumbing.Reference) {
	remote := git.NewRemote(nil, &config.RemoteConfig{URLs: []string{
		repoUrl,
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

func FindRepoHeadSha(repoUrl string) (string, error) {
	remote := git.NewRemote(nil, &config.RemoteConfig{
		Name: "origin",
		URLs: []string{repoUrl},
	})
	refs, err := remote.List(&git.ListOptions{})
	if err != nil {
		log.Fatalf("failed to list remote refs: %v", err)
	}

	var headTarget *plumbing.ReferenceName
	for _, ref := range refs {
		if ref.Name() == plumbing.HEAD {
			target := ref.Target()
			headTarget = &target
			break
		}
	}

	if headTarget == nil {
		return "", fmt.Errorf("HEAD reference not found")
	}

	// Now find the actual reference it points to (the default branch's tip)
	for _, ref := range refs {
		if ref.Name() == *headTarget {
			return ref.Hash().String(), nil
		}
	}

	return "", fmt.Errorf("could not resolve HEAD target %s", *headTarget)
}

func FindBranchHeadSha(repoUrl, branch string) (string, error) {
	// Check if the branch is already a valid SHA
	if plumbing.IsHash(branch) {
		return branch, nil
	}

	remote := git.NewRemote(nil, &config.RemoteConfig{
		Name: "origin",
		URLs: []string{repoUrl},
	})
	refs, err := remote.List(&git.ListOptions{
		Timeout: 30,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list remote refs: %v", err)
	}

	branchRef := plumbing.NewBranchReferenceName(branch)

	for _, ref := range refs {
		refName := ref.Name()
		if refName == branchRef {
			return ref.Hash().String(), nil
		}
	}

	return "", fmt.Errorf("branch %s not found in remote", branch)
}

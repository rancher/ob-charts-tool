package git

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/mallardduck/ob-charts-tool/internal/util"
	log "github.com/sirupsen/logrus"
	"strings"
)

func VerifyTagExists(repo string, tag string) (bool, string) {
	remote := git.NewRemote(nil, &config.RemoteConfig{URLs: []string{
		repo,
	}})
	refs, err := remote.List(&git.ListOptions{})
	if err != nil {
		log.Fatalf("Error listing remote refs: %v", err)
	}

	// Check if the reference exists
	found := false
	var hash string
	for _, ref := range refs {
		if ref.Name().String() == "refs/tags/"+tag {
			found = true
			hash = ref.Hash().String()
			fmt.Printf("Found reference: %s (%s)\n", ref.Name(), ref.Hash())
			break
		}
	}

	return found, hash
}

func FindTagsMatching(repo string, tagPartial string) (bool, []*plumbing.Reference) {
	remote := git.NewRemote(nil, &config.RemoteConfig{URLs: []string{
		repo,
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

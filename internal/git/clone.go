package git

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/rancher/ob-charts-tool/internal/git/cache"
)

type RepoConfigParams struct {
	Name   string
	URL    string
	Branch string
	Head   plumbing.Hash
}

func ShallowCloneRepository(repoConfig RepoConfigParams, destinationDir string) (plumbing.Hash, error) {
	repoClone, err := git.PlainClone(destinationDir, false, &git.CloneOptions{
		URL:               repoConfig.URL,
		Depth:             1,
		RecurseSubmodules: git.NoRecurseSubmodules,
		Progress:          nil,
	})
	if err != nil {
		return plumbing.ZeroHash, err
	}
	head, err := repoClone.Head()
	if err != nil {
		return plumbing.ZeroHash, err
	}
	return head.Hash(), nil
}

func CachedShallowCloneRepository(repoConfig RepoConfigParams, destinationDir string) (plumbing.Hash, error) {
	cacheManager, err := cache.GetGitCacheManager()
	if err != nil {
		return plumbing.ZeroHash, err
	}
	// First find and check cache path,
	// If not exist, create in cache path first.
	useShallow := true
	hasCache, err := cacheManager.HasRepoCache(repoConfig.Name, repoConfig.Head.String(), &useShallow)
	if !hasCache && err == nil {
		_, cloneErr := ShallowCloneRepository(repoConfig, cacheManager.GetRepoDir(repoConfig.Name, repoConfig.Head.String(), &useShallow))
		if cloneErr != nil {
			return plumbing.ZeroHash, cloneErr
		}
	} else if err != nil {
		return plumbing.ZeroHash, err
	}
	// Then we will copy from cache to dest.
	return cacheManager.CopyCacheTo(repoConfig.Name, repoConfig.Head, destinationDir, &useShallow)
}

func CloneRepository(repoConfig RepoConfigParams, destinationDir string) (plumbing.Hash, error) {
	repoClone, err := git.PlainClone(destinationDir, false, &git.CloneOptions{
		URL:               repoConfig.URL,
		RecurseSubmodules: git.NoRecurseSubmodules,
		Progress:          nil,
	})
	if err != nil {
		return plumbing.ZeroHash, err
	}
	head, err := repoClone.Head()
	if err != nil {
		return plumbing.ZeroHash, err
	}
	return head.Hash(), nil
}

func CachedCloneRepository(repoConfig RepoConfigParams, destinationDir string) (plumbing.Hash, error) {
	cacheManager, err := cache.GetGitCacheManager()
	if err != nil {
		return plumbing.ZeroHash, err
	}
	// First find and check cache path,
	// If not exist, create in cache path first.
	useShallow := false
	hasCache, err := cacheManager.HasRepoCache(repoConfig.Name, repoConfig.Head.String(), &useShallow)
	if !hasCache && err == nil {
		_, cloneErr := CloneRepository(repoConfig, cacheManager.GetRepoDir(repoConfig.Name, repoConfig.Head.String(), &useShallow))
		if cloneErr != nil {
			return plumbing.ZeroHash, cloneErr
		}
	} else if err != nil {
		return plumbing.ZeroHash, err
	}
	// Then we will copy from cache to dest.
	return cacheManager.CopyCacheTo(repoConfig.Name, repoConfig.Head, destinationDir, &useShallow)
}

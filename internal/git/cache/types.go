package cache

import (
	"errors"
	"github.com/go-git/go-git/v5/plumbing"
	cp "github.com/otiai10/copy"
	"github.com/rancher/ob-charts-tool/internal/util"
	"os"
	"path/filepath"
	"sync"
)

var (
	once          sync.Once
	cacheInstance *GitCacheManager
)

type GitCacheManager struct {
	rootDir string
}

func (gitCache *GitCacheManager) GetCacheDir() string {
	return gitCache.rootDir
}

func (gitCache *GitCacheManager) GetRepoDir(repoName string, sha string, useShallow *bool) string {
	rootDir := util.GetRepoCloneDir(gitCache.rootDir, repoName, sha)

	if useShallow != nil && *useShallow {
		return filepath.Join(rootDir, "shallow")
	}
	return filepath.Join(rootDir, repoName)
}

func (gitCache *GitCacheManager) HasRepoCache(repoName string, sha string, useShallow *bool) (bool, error) {
	repoPath := gitCache.GetRepoDir(repoName, sha, useShallow)
	if _, err := os.Stat(repoPath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (gitCache *GitCacheManager) CopyCacheTo(repoName string, sha plumbing.Hash, destinationDir string, useShallow *bool) (plumbing.Hash, error) {
	if has, _ := gitCache.HasRepoCache(repoName, sha.String(), useShallow); !has {
		return plumbing.ZeroHash, errors.New("repo not initialized")
	}

	repoCacheDir := gitCache.GetRepoDir(repoName, sha.String(), useShallow)
	// Verify destinationDir is empty or doesn't exist yet
	destErr := checkDestDirState(destinationDir)
	if destErr != nil {
		return plumbing.ZeroHash, destErr
	}

	// If we know the dest is in OK shape, then copy source from cache...
	copyErr := cp.Copy(repoCacheDir, destinationDir)
	if copyErr != nil {
		return plumbing.ZeroHash, copyErr
	}

	// On success we will
	return sha, nil
}

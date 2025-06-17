package util

import (
	"fmt"
	"path/filepath"
)

func GetRepoCloneDir(baseDir string, repoName string, sha string) string {
	return filepath.Join(baseDir, fmt.Sprintf("%s@%s", repoName, sha))
}

package util

import (
	"fmt"
	"os"
	"path/filepath"
)

func GetCacheDir(appName string) (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(cacheDir, appName), nil
}

func IsFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false // e.g., file doesn't exist
	}
	return info.Mode().IsRegular()
}

func GetRelativePath(cwd, filePath string) (string, error) {
	cleanCwd := filepath.Clean(cwd)
	cleanFilePath := filepath.Clean(filePath)

	relPath, err := filepath.Rel(cleanCwd, cleanFilePath)
	if err != nil {
		return "", fmt.Errorf("could not determine relative path for %s: %w", filePath, err)
	}

	return relPath, nil
}

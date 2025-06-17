package util

import (
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

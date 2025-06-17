package cache

import (
	"errors"
	"fmt"
	"os"
)

func SetupGitCacheManager(cacheRoot string) {
	once.Do(func() {
		if err := os.MkdirAll(cacheRoot, 0o755); err != nil {
			panic(fmt.Sprint("unable to create cache root: ", err))
		}

		cacheInstance = &GitCacheManager{
			rootDir: cacheRoot,
		}
	})
}

func GetGitCacheManager() (*GitCacheManager, error) {
	if cacheInstance == nil {
		return nil, errors.New("git cache manager not initialized")
	}

	return cacheInstance, nil
}

// checkDestDirState will error if the destDir is anything but: a) non-existent, or b) empty
func checkDestDirState(destDir string) error {
	info, err := os.Stat(destDir)
	if err != nil {
		// When the dir doesn't exist we can continue
		if os.IsNotExist(err) {
			// Directory doesn't exist, create it
			if err := os.MkdirAll(destDir, 0755); err != nil {
				return fmt.Errorf("failed to create destination directory: %w", err)
			}

			return nil
		}

		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("destination exists but is not a directory: %s", destDir)
	}
	entries, err := os.ReadDir(destDir)
	if err != nil {
		return fmt.Errorf("error reading destination directory: %w", err)
	}
	if len(entries) > 0 {
		return fmt.Errorf("destination directory is not empty: %s", destDir)
	}
	return nil
}

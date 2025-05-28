package fs

import (
	"bufio"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func searchDirForString(path, search string) (bool, error) {
	found := false
	err := filepath.WalkDir(path, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			logrus.Fatalf("Error accessing path %s: %v", path, err)
			return nil
		}

		if !info.IsDir() {
			contains, err := findStringInFile(path, search)
			if err != nil {
				logrus.Fatalf("Error accessing path %s: %v", path, err)
				return nil
			}
			if contains {
				found = true
				return fs.SkipAll
			}
		}

		return nil
	})
	if err != nil {
		logrus.Fatal(err)
	}

	return found, nil
}

func findStringInFile(path, search string) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsPermission(err) {
			return false, nil
		}
		logrus.Fatal(err)
		return false, fmt.Errorf("error reading file %s: %w", path, err)
	}

	return strings.Contains(string(content), search), nil
}

type SearchResult struct {
	Name string
	Path string
}

// FindSubdirsWithStringInFile will return the (first level) subdirs where the string can be found in files
func FindSubdirsWithStringInFile(path, search string) ([]string, error) {
	var result []string

	subdirs, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, subdir := range subdirs {
		if subdir.Type() != fs.ModeDir {
			continue
		}
		subChartPath := filepath.Join(path, subdir.Name(), "charts")
		foundString, err := searchDirForString(subChartPath, search)
		if err != nil {
			return result, err
		}
		if foundString {
			result = append(result, subdir.Name())
		}
	}

	return result, nil
}

type SearchReference struct {
	Content string `yaml:"content"`
	File    string `yaml:"file"`
	Line    int    `yaml:"line"`
}

func FindReferencesIn(path, search string) ([]SearchReference, error) {
	var result []SearchReference

	// Check if the provided path is a directory
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("could not get file info for path %s: %w", path, err)
	}
	if !fileInfo.IsDir() {
		// If it's a file, search within the single file
		references, err := findReferencesInFile(path, search)
		if err != nil {
			return nil, fmt.Errorf("error searching in file %s: %w", path, err)
		}
		result = append(result, references...)
		return result, nil
	}

	// Walk through the directory and its subdirectories
	err = filepath.WalkDir(path, func(filePath string, info os.DirEntry, err error) error {
		if err != nil {
			// Propagate the error
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Search within the file
		references, err := findReferencesInFile(filePath, search)
		if err != nil {
			// Log the error and continue with the walk, or return the error to stop the walk
			// For this implementation, we'll return the error to stop the walk.
			return fmt.Errorf("error searching in file %s: %w", filePath, err)
		}
		result = append(result, references...)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking the directory %s: %w", path, err)
	}

	return result, nil
}

// findReferencesInFile searches for a string within a single file.
func findReferencesInFile(filePath, search string) ([]SearchReference, error) {
	var fileReferences []SearchReference

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not open file %s: %w", filePath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		if strings.Contains(line, search) {
			fileReferences = append(fileReferences, SearchReference{
				Content: line,
				File:    filePath,
				Line:    lineNumber,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file %s line by line: %w", filePath, err)
	}

	return fileReferences, nil
}

package branchverifycheck

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-billy/v5/util"
)

// newPackageInfo creates a PackageInfo from a path to a package.yaml file
// packageYAMLPath should be the full path to a package.yaml file
// repoRoot should be the repository root path
func newPackageInfo(packageYAMLPath string, repoRoot string) PackageInfo {
	// Get the relative path from the packages directory
	packagesDir := filepath.Join(repoRoot, "packages")
	relPath, err := filepath.Rel(packagesDir, packageYAMLPath)
	if err != nil {
		return PackageInfo{}
	}

	// Remove the "package.yaml" filename
	dir := filepath.Dir(relPath)

	// Split the path to analyze structure
	parts := strings.Split(dir, string(filepath.Separator))

	var name string
	var versionDir string
	var rootDir string
	var fullPath string

	switch len(parts) {
	case 1:
		// packages/<name>/package.yaml
		name = parts[0]
		versionDir = ""
		rootDir = filepath.Join(packagesDir, name)
		fullPath = name
	case 2:
		// packages/<name>/<version>/package.yaml
		name = parts[0]
		versionDir = parts[1]
		rootDir = filepath.Join(packagesDir, name, parts[1])
		fullPath = name + "/" + versionDir
	case 3:
		// packages/<name>/<version>/<name>/package.yaml
		if parts[0] != parts[2] {
			return PackageInfo{}
		}
		name = parts[0]
		versionDir = parts[1]
		rootDir = filepath.Join(packagesDir, name, parts[1])
		fullPath = name + "/" + versionDir
	default:
		// Unexpected structure
		return PackageInfo{}
	}

	return PackageInfo{
		Name:            name,
		VersionDir:      versionDir,
		FullPath:        fullPath,
		PackageYAMLPath: packageYAMLPath,
		rootDir:         rootDir,
	}
}

// FindAllPackages takes a path to "rancher charts" style repo root and then finds all package indexes
func FindAllPackages(inPath string) map[string]PackageInfo {
	// Create a billy filesystem rooted at the repo
	fs := osfs.New(inPath)

	packages := make(map[string]PackageInfo)

	// Walk the packages directory looking for package.yaml files
	err := util.Walk(fs, "packages", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a file or not named package.yaml
		if info.IsDir() || info.Name() != "package.yaml" {
			return nil
		}

		// Convert to absolute path for newPackageInfo
		absPath := filepath.Join(inPath, path)

		// Create a PackageInfo for this package.yaml
		pkg := newPackageInfo(absPath, inPath)
		if pkg.Name != "" {
			packages[pkg.Name] = pkg
		}

		return nil
	})

	if err != nil {
		return nil
	}

	return packages
}

package branchverifycheck

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/go-git/go-git/v5"
	gitpkg "github.com/rancher/ob-charts-tool/internal/git"
	"gopkg.in/yaml.v3"
)

// CheckIsGitRepo verifies the path is a valid git repository.
func CheckIsGitRepo(path string) CheckResult {
	check := CheckResult{
		Name:     "Git Repository",
		Critical: true,
	}

	if !gitpkg.VerifyDirIsGitRepo(path) {
		check.Passed = false
		check.Message = "Path is not a git repository"
		return check
	}

	check.Passed = true
	check.Message = "Path is a valid git repository"
	return check
}

// CheckHasObTeamChartsRemote verifies the repo has the canonical rancher/ob-team-charts remote.
// This remote is to be treated as the "upstream" repository.
// Accepts both HTTPS and SSH formats:
//   - https://github.com/rancher/ob-team-charts.git
//   - https://github.com/rancher/ob-team-charts
//   - git@github.com:rancher/ob-team-charts.git
func CheckHasObTeamChartsRemote(repo *git.Repository) CheckResult {
	check := CheckResult{
		Name:     "Upstream Repository",
		Critical: true,
	}

	remotes, err := gitpkg.GetRemoteURLs(repo)
	if err != nil {
		check.Passed = false
		check.Message = fmt.Sprintf("Failed to get remotes: %v", err)
		return check
	}

	for remoteName, urls := range remotes {
		for _, url := range urls {
			if isCanonicalUpstream(url) {
				check.Passed = true
				check.Message = fmt.Sprintf("Found canonical upstream in remote '%s'", remoteName)
				return check
			}
		}
	}

	check.Passed = false
	check.Message = "No remote points to canonical upstream (rancher/ob-team-charts)"
	return check
}

// isCanonicalUpstream checks if a URL points to the canonical rancher/ob-team-charts repo.
func isCanonicalUpstream(url string) bool {
	// Normalize the URL for comparison
	normalized := strings.TrimSuffix(url, ".git")
	normalized = strings.ToLower(normalized)

	// Check HTTPS format: https://github.com/rancher/ob-team-charts
	if normalized == "https://github.com/rancher/ob-team-charts" {
		return true
	}

	// Check SSH format: git@github.com:rancher/ob-team-charts
	if normalized == "git@github.com:rancher/ob-team-charts" {
		return true
	}

	return false
}

// CheckOnFeatureBranch verifies we're on a feature branch, not main/master.
// Returns the branch name and the check result.
func CheckOnFeatureBranch(repo *git.Repository) (string, CheckResult) {
	check := CheckResult{
		Name:     "Branch Status",
		Critical: true,
	}

	branchName, err := gitpkg.FindRepoBranchName(repo)
	if err != nil {
		check.Passed = false
		check.Message = fmt.Sprintf("Failed to find branch name: %v", err)
		return "", check
	}

	if branchName == "main" || branchName == "master" {
		check.Passed = false
		check.Message = fmt.Sprintf("Currently on main branch '%s', should be on a feature branch", branchName)
		return branchName, check
	}

	check.Passed = true
	check.Message = fmt.Sprintf("On branch '%s'", branchName)
	return branchName, check
}

// CheckBranchCurrent checks if the branch is up-to-date with upstream/main.
func CheckBranchCurrent(refs *GitRefs, repo *git.Repository) CheckResult {
	check := CheckResult{
		Name:     "Branch Current with Upstream",
		Critical: false, // Soft fail - just a warning
	}

	if refs.MergeBaseCommit.Hash == refs.UpstreamCommit.Hash {
		check.Passed = true
		check.Message = "Branch is up-to-date with upstream/main"
		return check
	}

	// Count how many commits behind
	commitsBehind := CountCommitsBehind(repo, refs.UpstreamRef, refs.MergeBaseCommit.Hash)

	check.Passed = false
	if commitsBehind > 0 {
		check.Message = fmt.Sprintf("Branch is %d commit(s) behind upstream/main - consider rebasing to ensure version checks are accurate", commitsBehind)
	} else {
		check.Message = "Branch is behind upstream/main - consider rebasing to ensure version checks are accurate"
	}
	return check
}

// FindModifiedPackages finds which packages were modified in the branch.
// Compares HEAD to the merge-base to find only changes made on this branch.
func FindModifiedPackages(refs *GitRefs) ([]PackageInfo, CheckResult) {
	check := CheckResult{
		Name:     "Modified Packages",
		Critical: false, // Soft fail
	}

	headTree, err := refs.HeadCommit.Tree()
	if err != nil {
		check.Passed = false
		check.Message = fmt.Sprintf("Failed to get HEAD tree: %v", err)
		return nil, check
	}

	mergeBaseTree, err := refs.MergeBaseCommit.Tree()
	if err != nil {
		check.Passed = false
		check.Message = fmt.Sprintf("Failed to get merge-base tree: %v", err)
		return nil, check
	}

	changes, err := mergeBaseTree.Diff(headTree)
	if err != nil {
		check.Passed = false
		check.Message = fmt.Sprintf("Failed to diff trees: %v", err)
		return nil, check
	}

	// Extract modified packages from the diff
	// We want to detect changes at the version level (e.g., "rancher-monitoring/77.9")
	packagesMap := make(map[string]bool)
	for _, change := range changes {
		path := change.To.Name
		if change.To.Name == "" {
			path = change.From.Name
		}

		// Check if it's in packages/ directory
		if strings.HasPrefix(path, "packages/") {
			parts := strings.Split(path, "/")
			if len(parts) >= 3 {
				// parts[0] = "packages", parts[1] = package name, parts[2] = version
				packageWithVersion := parts[1] + "/" + parts[2]
				packagesMap[packageWithVersion] = true
			}
		}
	}

	// Convert to PackageInfo slice
	var packages []PackageInfo
	for fullPath := range packagesMap {
		parts := strings.Split(fullPath, "/")
		if len(parts) == 2 {
			packages = append(packages, PackageInfo{
				FullPath:   fullPath,
				Name:       parts[0],
				VersionDir: parts[1],
			})
		}
	}

	if len(packages) == 0 {
		// No packages modified is not a failure - branch might modify docs, CI, etc.
		check.Passed = true
		check.Message = "No packages modified in this branch (nothing to verify)"
		return packages, check
	}

	if len(packages) > 1 {
		var names []string
		for _, p := range packages {
			names = append(names, p.FullPath)
		}
		check.Passed = false
		check.Message = fmt.Sprintf("Multiple package versions modified: %v (should modify only one)", names)
	} else {
		check.Passed = true
		check.Message = fmt.Sprintf("Single package version modified: %s", packages[0].FullPath)
	}

	return packages, check
}

// CheckSequentialVersion verifies the package version is sequential.
// Applies some package specific logic to identify the correct `package.yaml` file.
// Reads the version from package.yaml, then verifies the previous version (n-1) exists
// in the built charts directory.
//
// For packages with -rancher.X suffix: verifies -rancher.(X-1) exists
// For packages without rancher suffix: verifies a lower version exists (simple semver)
func CheckSequentialVersion(repoPath string, pkg PackageInfo) CheckResult {
	check := CheckResult{
		Name:     fmt.Sprintf("Sequential Version Check (%s)", pkg.FullPath),
		Critical: true,
	}

	// Determine package.yaml path based on package type
	var packageYAMLPath string
	if pkg.Name == "rancher-monitoring" {
		// rancher-monitoring has a subdirectory with the package name
		packageYAMLPath = filepath.Join(repoPath, "packages", pkg.Name, pkg.VersionDir, pkg.Name, "package.yaml")
	} else {
		// rancher-logging, rancher-project-monitoring, etc. have package.yaml at version root
		packageYAMLPath = filepath.Join(repoPath, "packages", pkg.Name, pkg.VersionDir, "package.yaml")
	}

	data, err := os.ReadFile(packageYAMLPath)
	if err != nil {
		check.Passed = false
		check.Message = fmt.Sprintf("Failed to read package.yaml: %v", err)
		return check
	}

	var pkgYAML PackageYAML
	if err := yaml.Unmarshal(data, &pkgYAML); err != nil {
		check.Passed = false
		check.Message = fmt.Sprintf("Failed to parse package.yaml: %v", err)
		return check
	}

	currentVersion := pkgYAML.Version

	// Get all built chart versions
	chartsDir := filepath.Join(repoPath, "charts", pkg.Name)
	builtVersions, err := os.ReadDir(chartsDir)
	if err != nil {
		if os.IsNotExist(err) {
			// Charts directory doesn't exist - the chart should have been built
			check.Passed = false
			check.Message = fmt.Sprintf("Charts directory does not exist for %s - chart has not been built", pkg.Name)
			return check
		}
		check.Passed = false
		check.Message = fmt.Sprintf("Failed to read charts directory: %v", err)
		return check
	}

	// Build a set of existing version strings for quick lookup
	existingVersions := make(map[string]bool)
	for _, vDir := range builtVersions {
		if vDir.IsDir() {
			existingVersions[vDir.Name()] = true
		}
	}

	// First, verify the current version exists in built charts
	if !existingVersions[currentVersion] {
		check.Passed = false
		check.Message = fmt.Sprintf("Version %s not found in built charts - chart has not been built", currentVersion)
		return check
	}

	// If there's only one version and it matches current, this is the first version (new package)
	if len(existingVersions) == 1 {
		check.Passed = true
		check.Message = fmt.Sprintf("Version %s is first version for this package", currentVersion)
		return check
	}

	// Check if this is a rancher-suffixed version
	currentRancherRelease, rancherErr := extractRancherRelease(currentVersion)

	if rancherErr == nil {
		// Has rancher suffix - check that n-1 exists
		if currentRancherRelease == 1 {
			// This is -rancher.1, which is the first release for this base version
			// No previous rancher release to check for this base, this is valid
			check.Passed = true
			check.Message = fmt.Sprintf("Version %s is first rancher release for this base version", currentVersion)
			return check
		}

		// Build the expected previous version string
		baseVersion := strings.Split(currentVersion, "-rancher.")[0]
		previousVersion := fmt.Sprintf("%s-rancher.%d", baseVersion, currentRancherRelease-1)

		if existingVersions[previousVersion] {
			check.Passed = true
			check.Message = fmt.Sprintf("Version %s is sequential (previous %s exists)", currentVersion, previousVersion)
			return check
		}

		check.Passed = false
		check.Message = fmt.Sprintf("Version %s is not sequential: previous version %s not found in built charts",
			currentVersion, previousVersion)
		return check
	}

	// No rancher suffix (e.g., rancher-project-monitoring) - verify there's at least one lower version
	currentVer, err := semver.NewVersion(currentVersion)
	if err != nil {
		check.Passed = false
		check.Message = fmt.Sprintf("Invalid version format: %s", currentVersion)
		return check
	}

	// Check if any existing version is lower than current (meaning this is a valid next version)
	hasLowerVersion := false
	var highestLower *semver.Version
	for vStr := range existingVersions {
		v, err := semver.NewVersion(vStr)
		if err != nil {
			continue
		}
		if v.LessThan(currentVer) {
			hasLowerVersion = true
			if highestLower == nil || v.GreaterThan(highestLower) {
				highestLower = v
			}
		}
	}

	if hasLowerVersion {
		check.Passed = true
		check.Message = fmt.Sprintf("Version %s is valid (previous version %s exists)", currentVersion, highestLower.String())
		return check
	}

	check.Passed = false
	check.Message = fmt.Sprintf("Version %s has no previous version in built charts", currentVersion)
	return check
}

func extractRancherRelease(version string) (int, error) {
	re := regexp.MustCompile(`-rancher\.(\d+)`)
	matches := re.FindStringSubmatch(version)
	if len(matches) < 2 {
		return 0, fmt.Errorf("no rancher release found in version: %s", version)
	}
	var release int
	_, err := fmt.Sscanf(matches[1], "%d", &release)
	return release, err
}

// RepoStatus holds the result of checking repository cleanliness.
type RepoStatus struct {
	IsClean       bool
	ModifiedFiles []string
	Error         error
}

// getRepoStatus checks if a repository has uncommitted changes.
// Returns the status including whether it's clean and any modified files.
func getRepoStatus(repo *git.Repository) RepoStatus {
	worktree, err := repo.Worktree()
	if err != nil {
		return RepoStatus{Error: fmt.Errorf("failed to get worktree: %w", err)}
	}

	status, err := worktree.Status()
	if err != nil {
		return RepoStatus{Error: fmt.Errorf("failed to get status: %w", err)}
	}

	if status.IsClean() {
		return RepoStatus{IsClean: true}
	}

	var modifiedFiles []string
	for file, fileStatus := range status {
		if fileStatus.Worktree != git.Unmodified || fileStatus.Staging != git.Unmodified {
			modifiedFiles = append(modifiedFiles, file)
		}
	}

	return RepoStatus{IsClean: false, ModifiedFiles: modifiedFiles}
}

// CheckRepoClean verifies the repository has no uncommitted changes.
func CheckRepoClean(repo *git.Repository) CheckResult {
	check := CheckResult{
		Name:     "Repository Clean Before Build",
		Critical: true,
	}

	status := getRepoStatus(repo)

	if status.Error != nil {
		check.Passed = false
		check.Message = status.Error.Error()
		return check
	}

	if !status.IsClean {
		check.Passed = false
		check.Message = fmt.Sprintf("Repository has uncommitted changes before build (cannot verify build cleanliness): %v", status.ModifiedFiles)
		return check
	}

	check.Passed = true
	check.Message = "Repository is clean before build"
	return check
}

// CheckBuildNoChanges runs the build and verifies no uncommitted changes are created.
func CheckBuildNoChanges(repoPath string, pkg PackageInfo) CheckResult {
	check := CheckResult{
		Name:     fmt.Sprintf("Build Check (%s)", pkg.FullPath),
		Critical: true,
	}

	// Run make charts with PACKAGE env var
	cmd := exec.Command("make", "charts")
	cmd.Dir = repoPath
	cmd.Env = append(os.Environ(), fmt.Sprintf("PACKAGE=%s", pkg.Name))

	output, err := cmd.CombinedOutput()
	if err != nil {
		check.Passed = false
		check.Message = fmt.Sprintf("Build failed: %v\nOutput: %s", err, string(output))
		return check
	}

	// Check for uncommitted changes after build
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		check.Passed = false
		check.Message = fmt.Sprintf("Failed to open repo after build: %v", err)
		return check
	}

	status := getRepoStatus(repo)

	if status.Error != nil {
		check.Passed = false
		check.Message = status.Error.Error()
		return check
	}

	if !status.IsClean {
		check.Passed = false
		check.Message = fmt.Sprintf("Build created uncommitted changes: %v", status.ModifiedFiles)
		return check
	}

	check.Passed = true
	check.Message = "Build successful with no uncommitted changes"
	return check
}

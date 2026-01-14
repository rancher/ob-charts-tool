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
			if gitpkg.IsGitHubRepoURL(url, "rancher", "ob-team-charts") {
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
func CheckBranchCurrent(refs *GitRefs, repo *git.Repository) (BranchInfo, CheckResult) {
	check := CheckResult{
		Name:     "Branch Current with Upstream",
		Critical: false, // Soft fail - just a warning
	}
	branchInfo := BranchInfo{
		Name:          refs.HeadRef.Name().Short(),
		CommitsBehind: 0,
	}

	if refs.MergeBaseCommit.Hash == refs.UpstreamCommit.Hash {
		check.Passed = true
		check.Message = "Branch is up-to-date with upstream/main"
		branchInfo.IsUpToDate = true
		return branchInfo, check
	}

	// Count how many commits behind
	branchInfo.CommitsBehind = CountCommitsBehind(repo, refs.UpstreamRef, refs.MergeBaseCommit.Hash)

	check.Passed = false
	branchInfo.IsUpToDate = false
	if branchInfo.CommitsBehind > 0 {
		check.Message = fmt.Sprintf("Branch is %d commit(s) behind upstream/main - consider rebasing to ensure version checks are accurate", branchInfo.CommitsBehind)
	} else {
		check.Message = "Branch is behind upstream/main - consider rebasing to ensure version checks are accurate"
	}
	return branchInfo, check
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
		check.Message = fmt.Sprintf("Multiple package versions modified: %v (recommend modifying only one)", names)
	} else {
		check.Passed = true
		check.Message = fmt.Sprintf("Single package version modified: %s", packages[0].FullPath)
	}

	return packages, check
}

// PackageVersionInfo contains version information extracted from a package.
type PackageVersionInfo struct {
	Version          string
	PackageYAMLPath  string
	ChartsDir        string
	ExistingVersions map[string]bool
}

// getPackageYAMLPath returns the path to package.yaml for a given package.
// Handles package-specific directory structures.
func getPackageYAMLPath(repoPath string, pkg PackageInfo) string {
	if pkg.Name == "rancher-monitoring" {
		// rancher-monitoring has a subdirectory with the package name
		return filepath.Join(repoPath, "packages", pkg.Name, pkg.VersionDir, pkg.Name, "package.yaml")
	}
	// rancher-logging, rancher-project-monitoring, etc. have package.yaml at version root
	return filepath.Join(repoPath, "packages", pkg.Name, pkg.VersionDir, "package.yaml")
}

// getPackageVersionInfo reads version info from package.yaml and existing built charts.
func getPackageVersionInfo(repoPath string, pkg PackageInfo) (*PackageVersionInfo, error) {
	info := &PackageVersionInfo{
		PackageYAMLPath: getPackageYAMLPath(repoPath, pkg),
		ChartsDir:       filepath.Join(repoPath, "charts", pkg.Name),
	}

	// Read and parse package.yaml
	data, err := os.ReadFile(info.PackageYAMLPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read package.yaml: %w", err)
	}

	var pkgYAML PackageYAML
	if err := yaml.Unmarshal(data, &pkgYAML); err != nil {
		return nil, fmt.Errorf("failed to parse package.yaml: %w", err)
	}
	info.Version = pkgYAML.Version

	// Get all built chart versions
	builtVersions, err := os.ReadDir(info.ChartsDir)
	if err != nil {
		if os.IsNotExist(err) {
			// Charts directory doesn't exist yet - that's ok, we'll return empty map
			info.ExistingVersions = make(map[string]bool)
			return info, nil
		}
		return nil, fmt.Errorf("failed to read charts directory: %w", err)
	}

	// Build a set of existing version strings for quick lookup
	info.ExistingVersions = make(map[string]bool)
	for _, vDir := range builtVersions {
		if vDir.IsDir() {
			info.ExistingVersions[vDir.Name()] = true
		}
	}

	return info, nil
}

// CheckChartBuilt verifies that a chart matching the package version exists in the charts directory.
func CheckChartBuilt(repoPath string, pkg PackageInfo) CheckResult {
	check := CheckResult{
		Name:     fmt.Sprintf("Chart Built (%s)", pkg.FullPath),
		Critical: true,
	}

	info, err := getPackageVersionInfo(repoPath, pkg)
	if err != nil {
		check.Passed = false
		check.Message = err.Error()
		return check
	}

	// Check if charts directory exists with any versions
	if len(info.ExistingVersions) == 0 {
		check.Passed = false
		check.Message = fmt.Sprintf("No charts exist for %s - chart has not been built", pkg.Name)
		return check
	}

	// Check if the current version exists in built charts
	if !info.ExistingVersions[info.Version] {
		check.Passed = false
		check.Message = fmt.Sprintf("Version %s not found in built charts - chart has not been built", info.Version)
		return check
	}

	check.Passed = true
	check.Message = fmt.Sprintf("Chart version %s exists in built charts", info.Version)
	return check
}

// CheckSequentialVersion verifies the package version is sequential (n-1 check).
// Reads the version from package.yaml, then verifies the previous version (n-1) exists
// in the built charts directory.
//
// For packages with -rancher.X suffix: verifies -rancher.(X-1) exists
// For packages without rancher suffix: verifies a lower version exists (simple semver)
func CheckSequentialVersion(repoPath string, pkg PackageInfo) CheckResult {
	check := CheckResult{
		Name:     fmt.Sprintf("Sequential Version (%s)", pkg.FullPath),
		Critical: true,
	}

	info, err := getPackageVersionInfo(repoPath, pkg)
	if err != nil {
		check.Passed = false
		check.Message = err.Error()
		return check
	}

	// If there are no existing versions or only the current version, this is the first version
	versionsExcludingCurrent := 0
	for v := range info.ExistingVersions {
		if v != info.Version {
			versionsExcludingCurrent++
		}
	}
	if versionsExcludingCurrent == 0 {
		check.Passed = true
		check.Message = fmt.Sprintf("Version %s is first version for this package", info.Version)
		return check
	}

	// Check if this is a rancher-suffixed version
	currentRancherRelease, rancherErr := extractRancherRelease(info.Version)

	if rancherErr == nil {
		// Has rancher suffix - check that n-1 exists
		if currentRancherRelease == 1 {
			// This is -rancher.1, which is the first release for this base version
			// No previous rancher release to check for this base, this is valid
			check.Passed = true
			check.Message = fmt.Sprintf("Version %s is first rancher release for this base version", info.Version)
			return check
		}

		// Build the expected previous version string
		baseVersion := strings.Split(info.Version, "-rancher.")[0]
		previousVersion := fmt.Sprintf("%s-rancher.%d", baseVersion, currentRancherRelease-1)

		if info.ExistingVersions[previousVersion] {
			check.Passed = true
			check.Message = fmt.Sprintf("Version %s is sequential (previous %s exists)", info.Version, previousVersion)
			return check
		}

		check.Passed = false
		check.Message = fmt.Sprintf("Version %s is not sequential: previous version %s not found in built charts",
			info.Version, previousVersion)
		return check
	}

	// No rancher suffix (e.g., rancher-project-monitoring) - verify there's at least one lower version
	currentVer, err := semver.NewVersion(info.Version)
	if err != nil {
		check.Passed = false
		check.Message = fmt.Sprintf("Invalid version format: %s", info.Version)
		return check
	}

	// Check if any existing version is lower than current (meaning this is a valid next version)
	hasLowerVersion := false
	var highestLower *semver.Version
	for vStr := range info.ExistingVersions {
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
		check.Message = fmt.Sprintf("Version %s is valid (previous version %s exists)", info.Version, highestLower.String())
		return check
	}

	check.Passed = false
	check.Message = fmt.Sprintf("Version %s has no previous version in built charts", info.Version)
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

	modifiedFiles := make([]string, 0, len(status))
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

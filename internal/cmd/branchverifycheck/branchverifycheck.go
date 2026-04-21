package branchverifycheck

import (
	"fmt"

	"github.com/go-git/go-git/v5"
)

// globalCheck prints msg, registers check globally, prints passMsg/failMsg, and returns Passed.
func globalCheck(p *ProgressPrinter, r *VerificationResult, msg string, check CheckResult, passMsg, failMsg string) bool {
	p.Print(msg)
	r.AddGlobalCheck(check)
	if check.Passed {
		p.Println(passMsg)
	} else {
		p.Println(failMsg)
	}
	return check.Passed
}

// packageCheck prints msg, registers check on the package result, and prints the outcome.
// On pass, prints "OK - <check.Message>" so developers see a brief status without needing JSON output.
// On fail, prints failMsg (e.g. "FAILED" or "WARN").
func packageCheck(p *ProgressPrinter, pr *PackageResult, msg string, check CheckResult, failMsg string) {
	p.Print(msg)
	pr.AddCheck(check)
	if check.Passed {
		p.Println("OK - " + check.Message)
	} else {
		p.Println(failMsg)
	}
}

// VerifyBranch takes a file path containing the ob-team-charts repo and verifies the branch state.
// It specifically is a tool to analyze the branch not the chart itself.
//
// Verification steps:
// - Basic checks: is path git repo, is upstream containing `ob-team-charts`, is on branch, is branch not main
// - Verify branch only modifies a single Package (soft-fail/warning)
// - Verify that the package+chart created is sequential to the last built chart (hard fail)
// - If chart build scripts are run targeting the modified package, no uncommitted changes are found (hard fail)
func VerifyBranch(path string, jsonOutput bool) (*VerificationResult, error) {
	result := &VerificationResult{
		Success:        true,
		GlobalChecks:   []CheckResult{},
		PackageResults: []PackageResult{},
	}

	// Use progress printer from output.go
	progress := NewProgressPrinter(jsonOutput)

	progress.Println("Starting branch verification...")
	progress.Println("")

	// ============================================
	// GLOBAL CHECKS - Run once for the whole branch
	// ============================================

	// Step 1: Check if path is a git repository
	progress.Print("Checking if path is a git repository... ")
	gitRepoCheck := CheckIsGitRepo(path)
	result.AddGlobalCheck(gitRepoCheck)
	if !gitRepoCheck.Passed {
		progress.Println("FAILED")
		return result, fmt.Errorf("path is not a git repository")
	}
	progress.Println("OK")

	// Open the repository
	repo, err := git.PlainOpen(path)
	if err != nil {
		return result, fmt.Errorf("failed to open git repo: %w", err)
	}

	// Step 2: Check upstream remote
	globalCheck(progress, result, "Checking upstream repository... ", CheckHasObTeamChartsRemote(repo), "OK", "FAILED")

	// Step 3: Check branch status (not on main/master)
	branchName, branchCheck := CheckOnFeatureBranch(repo)
	globalCheck(progress, result, "Checking branch status... ", branchCheck,
		fmt.Sprintf("OK (on branch '%s')", branchName), "FAILED")

	// Step 4: Get git refs (ensures upstream remote, fetches, finds merge-base)
	progress.Print("Setting up upstream remote and fetching... ")
	refs, err := GetGitRefs(repo, path)
	// Ensure cleanup of any temporary remote we created
	defer CleanupToolRemote(repo)
	if err != nil {
		progress.Println("FAILED")
		result.AddGlobalCheck(CheckResult{
			Name:     "Git References",
			Passed:   false,
			Message:  fmt.Sprintf("Failed to get git refs: %v", err),
			Critical: true,
		})
		// Continue to output results
		outputResults(result, branchName, jsonOutput)
		return result, err
	}
	progress.Println("OK")

	// Step 5: Check if branch is current with upstream
	branchInfo, currentCheck := CheckBranchCurrent(refs, repo)
	behindMsg := "WARN - Branch is behind main"
	if branchInfo.CommitsBehind != 0 {
		behindMsg = fmt.Sprintf("WARN - Branch is %d commit(s) behind main", branchInfo.CommitsBehind)
	}
	globalCheck(progress, result, "Checking if branch is current with upstream... ", currentCheck, "OK", behindMsg)

	// Step 6: Find modified packages
	progress.Print("Finding modified packages... ")
	packages, packagesCheck := FindModifiedPackages(refs)
	result.AddGlobalCheck(packagesCheck)
	if len(packages) > 0 {
		progress.Printf("OK (found: %v)\n", packageNames(packages))
	} else {
		progress.Println("NONE")
	}
	progress.Println("")

	// ============================================
	// PER-PACKAGE CHECKS - Run for each modified package
	// ============================================

	for _, pkg := range packages {
		pkgResult := result.GetOrCreatePackageResult(pkg)

		packageCheck(progress, pkgResult, fmt.Sprintf("Checking sequential version for %s... ", pkg.FullPath),
			CheckSequentialVersion(path, pkg), "FAILED")
		packageCheck(progress, pkgResult, fmt.Sprintf("Checking chart built for %s... ", pkg.FullPath),
			CheckChartBuilt(path, pkg), "FAILED")
	}

	// ============================================
	// GLOBAL CHECK - Repository cleanliness before build
	// ============================================

	repoIsClean := globalCheck(progress, result, "Checking repository is clean before build... ",
		CheckRepoClean(repo), "OK", "DIRTY (skipping build checks)")

	// ============================================
	// PER-PACKAGE CHECK - Build verification (only if repo was clean)
	// ============================================

	if repoIsClean {
		for _, pkg := range packages {
			pkgResult := result.GetOrCreatePackageResult(pkg)

			packageCheck(progress, pkgResult, fmt.Sprintf("Running build check for %s (this may take a while)... ", pkg.FullPath),
				CheckBuildNoChanges(path, pkg), "FAILED")
			packageCheck(progress, pkgResult, fmt.Sprintf("Running package image check for %s (this may take a while)... ", pkg.FullPath),
				CheckPackageImages(path, pkg), "FAILED")
			packageCheck(progress, pkgResult, fmt.Sprintf("Checking subchart appVersion tags for %s... ", pkg.FullPath),
				CheckSubchartAppVersionTags(path, pkg), "WARN")
		}
	}

	// Output results
	outputResults(result, branchName, jsonOutput)

	// Determine overall success
	result.Success = !result.HasCriticalFailure()

	return result, nil
}

// outputResults handles outputting results in the appropriate format
func outputResults(result *VerificationResult, branchName string, jsonOutput bool) {
	if jsonOutput {
		OutputJSON(result)
	} else {
		OutputHuman(result, branchName)
	}
}

// packageNames extracts the FullPath from a slice of PackageInfo for display
func packageNames(packages []PackageInfo) []string {
	names := make([]string, len(packages))
	for i, pkg := range packages {
		names[i] = pkg.FullPath
	}
	return names
}

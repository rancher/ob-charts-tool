package branchverifycheck

import (
	"fmt"

	"github.com/go-git/go-git/v5"
)

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
	progress.Print("Checking upstream repository... ")
	upstreamCheck := CheckHasObTeamChartsRemote(repo)
	result.AddGlobalCheck(upstreamCheck)
	if !upstreamCheck.Passed {
		progress.Println("FAILED")
	} else {
		progress.Println("OK")
	}

	// Step 3: Check branch status (not on main/master)
	progress.Print("Checking branch status... ")
	branchName, branchCheck := CheckOnFeatureBranch(repo)
	result.AddGlobalCheck(branchCheck)
	if !branchCheck.Passed {
		progress.Println("FAILED")
	} else {
		progress.Printf("OK (on branch '%s')\n", branchName)
	}

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
	progress.Print("Checking if branch is current with upstream... ")
	currentCheck := CheckBranchCurrent(refs, repo)
	result.AddGlobalCheck(currentCheck)
	if !currentCheck.Passed {
		progress.Println("WARN")
	} else {
		progress.Println("OK")
	}

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

		// Check sequential versioning
		progress.Printf("Checking sequential version for %s... ", pkg.FullPath)
		versionCheck := CheckSequentialVersion(path, pkg)
		pkgResult.AddCheck(versionCheck)
		if versionCheck.Passed {
			progress.Println("OK")
		} else {
			progress.Println("FAILED")
		}

		// Check that chart has been built
		progress.Printf("Checking chart built for %s... ", pkg.FullPath)
		chartCheck := CheckChartBuilt(path, pkg)
		pkgResult.AddCheck(chartCheck)
		if chartCheck.Passed {
			progress.Println("OK")
		} else {
			progress.Println("FAILED")
		}
	}

	// ============================================
	// GLOBAL CHECK - Repository cleanliness before build
	// ============================================

	progress.Print("Checking repository is clean before build... ")
	cleanCheck := CheckRepoClean(repo)
	result.AddGlobalCheck(cleanCheck)
	repoIsClean := cleanCheck.Passed
	if !repoIsClean {
		progress.Println("DIRTY (skipping build checks)")
	} else {
		progress.Println("OK")
	}

	// ============================================
	// PER-PACKAGE CHECK - Build verification (only if repo was clean)
	// ============================================

	if repoIsClean {
		for _, pkg := range packages {
			pkgResult := result.GetOrCreatePackageResult(pkg)

			progress.Printf("Running build check for %s (this may take a while)... ", pkg.FullPath)
			buildCheck := CheckBuildNoChanges(path, pkg)
			pkgResult.AddCheck(buildCheck)
			if buildCheck.Passed {
				progress.Println("OK")
			} else {
				progress.Println("FAILED")
			}
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

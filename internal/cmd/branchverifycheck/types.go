package branchverifycheck

import (
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// CheckResult represents the result of a single verification check
type CheckResult struct {
	Name     string `json:"name"`
	Passed   bool   `json:"passed"`
	Message  string `json:"message"`
	Critical bool   `json:"critical"` // If true, failure should exit with error
}

// PackageResult groups all check results for a single package
type PackageResult struct {
	Package PackageInfo   `json:"package"`
	Checks  []CheckResult `json:"checks"`
}

// AddCheck appends a check result to the package result
func (p *PackageResult) AddCheck(check CheckResult) {
	p.Checks = append(p.Checks, check)
}

// HasCriticalFailure returns true if any critical check failed
func (p *PackageResult) HasCriticalFailure() bool {
	for _, check := range p.Checks {
		if !check.Passed && check.Critical {
			return true
		}
	}
	return false
}

// VerificationResult represents the overall verification result
type VerificationResult struct {
	Success        bool            `json:"success"`
	GlobalChecks   []CheckResult   `json:"globalChecks"`
	PackageResults []PackageResult `json:"packageResults"`
}

// AddGlobalCheck appends a check result to the global checks
func (r *VerificationResult) AddGlobalCheck(check CheckResult) {
	r.GlobalChecks = append(r.GlobalChecks, check)
}

// GetOrCreatePackageResult returns the PackageResult for the given package,
// creating one if it doesn't exist
func (r *VerificationResult) GetOrCreatePackageResult(pkg PackageInfo) *PackageResult {
	for i := range r.PackageResults {
		if r.PackageResults[i].Package.FullPath == pkg.FullPath {
			return &r.PackageResults[i]
		}
	}
	// Create new package result
	r.PackageResults = append(r.PackageResults, PackageResult{
		Package: pkg,
		Checks:  []CheckResult{},
	})
	return &r.PackageResults[len(r.PackageResults)-1]
}

// HasCriticalFailure returns true if any critical check (global or package) failed
func (r *VerificationResult) HasCriticalFailure() bool {
	for _, check := range r.GlobalChecks {
		if !check.Passed && check.Critical {
			return true
		}
	}
	for _, pkgResult := range r.PackageResults {
		if pkgResult.HasCriticalFailure() {
			return true
		}
	}
	return false
}

// CountResults returns counts of passed, failed, and warning checks
func (r *VerificationResult) CountResults() (passed, failed, warnings int) {
	for _, check := range r.GlobalChecks {
		if check.Passed {
			passed++
		} else if check.Critical {
			failed++
		} else {
			warnings++
		}
	}
	for _, pkgResult := range r.PackageResults {
		for _, check := range pkgResult.Checks {
			if check.Passed {
				passed++
			} else if check.Critical {
				failed++
			} else {
				warnings++
			}
		}
	}
	return
}

// PackageYAML represents the structure of a package.yaml file
type PackageYAML struct {
	Version string `yaml:"version"`
}

// GitRefs holds the git references needed for verification
type GitRefs struct {
	// HeadRef is the current HEAD reference
	HeadRef *plumbing.Reference
	// HeadCommit is the commit at HEAD
	HeadCommit *object.Commit
	// UpstreamRef is the upstream main branch reference
	UpstreamRef *plumbing.Reference
	// UpstreamCommit is the commit at upstream main
	UpstreamCommit *object.Commit
	// MergeBaseCommit is the common ancestor between HEAD and upstream
	MergeBaseCommit *object.Commit
}

// BranchInfo holds information about the current branch
type BranchInfo struct {
	Name          string
	CommitsBehind int
	IsUpToDate    bool
}

// PackageInfo holds information about a modified package
type PackageInfo struct {
	// FullPath is the package with version (e.g., "rancher-monitoring/77.9")
	FullPath string `json:"fullPath"`
	// Name is just the package name (e.g., "rancher-monitoring")
	Name string `json:"name"`
	// VersionDir is the version directory (e.g., "77.9")
	VersionDir string `json:"versionDir"`
}

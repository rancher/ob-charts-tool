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

// VerificationResult represents the overall verification result
type VerificationResult struct {
	Success bool          `json:"success"`
	Checks  []CheckResult `json:"checks"`
}

// AddCheck appends a check result to the verification result
func (r *VerificationResult) AddCheck(check CheckResult) {
	r.Checks = append(r.Checks, check)
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
	FullPath string
	// Name is just the package name (e.g., "rancher-monitoring")
	Name string
	// VersionDir is the version directory (e.g., "77.9")
	VersionDir string
}

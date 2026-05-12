package branchverifycheck

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cr builds a minimal CheckResult for use in table-driven tests.
func cr(passed, critical bool) CheckResult {
	return CheckResult{Passed: passed, Critical: critical}
}

func TestPackageResult_HasCriticalFailure(t *testing.T) {
	cases := []struct {
		name   string
		checks []CheckResult
		want   bool
	}{
		{"empty checks", nil, false},
		{"single passed critical", []CheckResult{cr(true, true)}, false},
		{"single failed non-critical", []CheckResult{cr(false, false)}, false},
		{"single failed critical", []CheckResult{cr(false, true)}, true},
		{"failed non-critical + passed critical", []CheckResult{cr(false, false), cr(true, true)}, false},
		{"passed non-critical + failed critical", []CheckResult{cr(true, false), cr(false, true)}, true},
		{"multiple non-critical failures only", []CheckResult{cr(false, false), cr(false, false)}, false},
		{"one of three is failed critical", []CheckResult{cr(true, true), cr(false, false), cr(false, true)}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pr := PackageResult{Checks: tc.checks}
			got := pr.HasCriticalFailure()
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestVerificationResult_HasCriticalFailure(t *testing.T) {
	cases := []struct {
		name          string
		globalChecks  []CheckResult
		packageChecks []CheckResult // added as a single PackageResult
		want          bool
	}{
		{"empty", nil, nil, false},
		{"global failed critical", []CheckResult{cr(false, true)}, nil, true},
		{"global failed non-critical only", []CheckResult{cr(false, false)}, nil, false},
		{"global passed", []CheckResult{cr(true, true)}, nil, false},
		{"package failed critical", nil, []CheckResult{cr(false, true)}, true},
		{"package failed non-critical only", nil, []CheckResult{cr(false, false)}, false},
		{"global passes, package fails critical", []CheckResult{cr(true, true)}, []CheckResult{cr(false, true)}, true},
		{"all pass", []CheckResult{cr(true, true)}, []CheckResult{cr(true, true)}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &VerificationResult{GlobalChecks: tc.globalChecks}
			if len(tc.packageChecks) > 0 {
				r.PackageResults = []PackageResult{
					{Package: PackageInfo{FullPath: "pkg/1.0"}, Checks: tc.packageChecks},
				}
			}
			got := r.HasCriticalFailure()
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestVerificationResult_CountResults(t *testing.T) {
	cases := []struct {
		name          string
		globalChecks  []CheckResult
		packageChecks []CheckResult
		wantPassed    int
		wantFailed    int
		wantWarnings  int
	}{
		{"empty", nil, nil, 0, 0, 0},
		{"one global pass", []CheckResult{cr(true, true)}, nil, 1, 0, 0},
		{"one global critical fail", []CheckResult{cr(false, true)}, nil, 0, 1, 0},
		{"one global warning (non-critical fail)", []CheckResult{cr(false, false)}, nil, 0, 0, 1},
		{
			name:         "mixed global: 1 pass, 1 fail, 1 warn",
			globalChecks: []CheckResult{cr(true, true), cr(false, true), cr(false, false)},
			wantPassed:   1, wantFailed: 1, wantWarnings: 1,
		},
		{
			name:          "package checks contribute",
			packageChecks: []CheckResult{cr(true, false), cr(false, true), cr(false, false)},
			wantPassed:    1, wantFailed: 1, wantWarnings: 1,
		},
		{
			name:          "global + package combined",
			globalChecks:  []CheckResult{cr(true, true), cr(false, false)},
			packageChecks: []CheckResult{cr(false, true)},
			wantPassed:    1, wantFailed: 1, wantWarnings: 1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &VerificationResult{GlobalChecks: tc.globalChecks}
			if len(tc.packageChecks) > 0 {
				r.PackageResults = []PackageResult{
					{Package: PackageInfo{FullPath: "pkg/1.0"}, Checks: tc.packageChecks},
				}
			}
			passed, failed, warnings := r.CountResults()
			assert.Equal(t, tc.wantPassed, passed, "passed count")
			assert.Equal(t, tc.wantFailed, failed, "failed count")
			assert.Equal(t, tc.wantWarnings, warnings, "warnings count")
		})
	}
}

func TestVerificationResult_GetOrCreatePackageResult(t *testing.T) {
	r := &VerificationResult{}

	pkg1 := PackageInfo{FullPath: "pkg-a/1.0", Name: "pkg-a", VersionDir: "1.0"}
	pkg2 := PackageInfo{FullPath: "pkg-b/2.0", Name: "pkg-b", VersionDir: "2.0"}

	// First call creates entry
	pr1 := r.GetOrCreatePackageResult(pkg1)
	require.NotNil(t, pr1, "GetOrCreatePackageResult should not return nil")
	assert.Len(t, r.PackageResults, 1)

	// Same key returns same pointer (no duplicate)
	pr1again := r.GetOrCreatePackageResult(pkg1)
	assert.Same(t, pr1, pr1again, "should return same pointer for same key")
	assert.Len(t, r.PackageResults, 1, "should not create duplicate")

	// Different key creates new entry
	pr2 := r.GetOrCreatePackageResult(pkg2)
	require.NotNil(t, pr2, "GetOrCreatePackageResult should not return nil for second package")
	assert.Len(t, r.PackageResults, 2)
	assert.NotSame(t, pr1, pr2, "should return different pointer for different key")
}

func TestCheckDetails_Format(t *testing.T) {
	t.Run("BuildDiffDetails", func(t *testing.T) {
		d := &BuildDiffDetails{
			ModifiedFiles: []string{"charts/foo/1.0/values.yaml"},
			Diff:          "- old line\n+ new line",
		}
		out := d.Format()
		assert.Contains(t, out, "charts/foo/1.0/values.yaml", "should contain modified file")
		assert.Contains(t, out, "Diff:", "should contain 'Diff:' header")
		assert.Contains(t, out, "- old line", "should contain diff content")
	})

	t.Run("ImageCheckDetails", func(t *testing.T) {
		d := &ImageCheckDetails{
			InvalidImages: []InvalidImage{
				{Path: "image", Issues: []string{"registry=docker.io (expected \"\")"}},
			},
			FilesChecked: 3,
		}
		out := d.Format()
		assert.Contains(t, out, "1", "should contain invalid image count")
		assert.Contains(t, out, "3", "should contain files checked count")
		assert.Contains(t, out, "image", "should contain image path")
		assert.Contains(t, out, "registry=docker.io", "should contain issue description")
	})

	t.Run("SubchartTagCheckDetails", func(t *testing.T) {
		d := &SubchartTagCheckDetails{
			Mismatches: []SubchartTagMismatch{
				{
					SubchartName:  "grafana",
					ValuesKey:     "image.tag",
					ActualValue:   "9.0.0",
					ExpectedValue: "10.0.0",
				},
			},
		}
		out := d.Format()
		assert.Contains(t, out, "grafana", "should contain subchart name")
		assert.Contains(t, out, "9.0.0", "should contain actual value")
		assert.Contains(t, out, "10.0.0", "should contain expected value")
	})
}
